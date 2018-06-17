package domain

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/karrick/godirwalk"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Map maps domain names to D instances.
type Map map[string]*D

type domainsUpdater func(Map)

func (dm Map) addDomain(rootDomain, group, projectName string, config *domainConfig) {
	newDomain := &D{
		group:       group,
		projectName: projectName,
		config:      config,
	}

	var domainName string
	domainName = strings.ToLower(config.Domain)
	dm[domainName] = newDomain
}

func (dm Map) updateGroupDomain(rootDomain, group, projectName string, httpsOnly bool, private bool, accessControl bool, id uint64) {
	domainName := strings.ToLower(group + "." + rootDomain)
	groupDomain := dm[domainName]

	if groupDomain == nil {
		groupDomain = &D{
			group:    group,
			projects: make(projects),
		}
	}

	groupDomain.projects[projectName] = &project{
		HTTPSOnly:     httpsOnly,
		Private:       private,
		AccessControl: accessControl,
		ID:            id,
	}

	dm[domainName] = groupDomain
}

func (dm Map) readProjectConfig(rootDomain string, group, projectName string, config *domainsConfig) {
	if config == nil {
		// This is necessary to preserve the previous behaviour where a
		// group domain is created even if no config.json files are
		// loaded successfully. Is it safe to remove this?
		dm.updateGroupDomain(rootDomain, group, projectName, false, false, false, 0)
		return
	}

	dm.updateGroupDomain(rootDomain, group, projectName, config.HTTPSOnly, config.Private, config.AccessControl, config.ID)

	for _, domainConfig := range config.Domains {
		config := domainConfig // domainConfig is reused for each loop iteration
		if domainConfig.Valid(rootDomain) {
			dm.addDomain(rootDomain, group, projectName, &config)
		}
	}
}

func readProject(group, projectName string, fanIn chan<- jobResult) {
	if strings.HasPrefix(projectName, ".") {
		return
	}

	// Ignore projects that have .deleted in name
	if strings.HasSuffix(projectName, ".deleted") {
		return
	}

	if _, err := os.Lstat(filepath.Join(group, projectName, "public")); err != nil {
		return
	}

	// We read the config.json file _before_ fanning in, because it does disk
	// IO and it does not need access to the domains map.
	config := &domainsConfig{}
	if err := config.Read(group, projectName); err != nil {
		config = nil
	}

	fanIn <- jobResult{group: group, project: projectName, config: config}
}

func readProjects(group string, buf []byte, fanIn chan<- jobResult) {
	fis, err := godirwalk.ReadDirents(group, buf)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"group": group,
		}).Print("readdir failed")
		return
	}

	for _, project := range fis {
		// Ignore non directories
		if !project.IsDir() {
			continue
		}

		readProject(group, project.Name(), fanIn)
	}
}

type jobResult struct {
	group   string
	project string
	config  *domainsConfig
}

// ReadGroups walks the pages directory and populates dm with all the domains it finds.
func (dm Map) ReadGroups(rootDomain string) error {
	fis, err := godirwalk.ReadDirents(".", nil)
	if err != nil {
		return err
	}

	fanOutGroups := make(chan string)
	fanIn := make(chan jobResult)
	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)

		go func() {
			buf := make([]byte, 2*os.Getpagesize())

			for group := range fanOutGroups {
				started := time.Now()

				readProjects(group, buf, fanIn)

				log.WithFields(log.Fields{
					"group":    group,
					"duration": time.Since(started).Seconds(),
				}).Debug("Loaded projects for group")
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(fanIn)
	}()

	done := make(chan struct{})
	go func() {
		for result := range fanIn {
			dm.readProjectConfig(rootDomain, result.group, result.project, result.config)
		}

		close(done)
	}()

	for _, group := range fis {
		if !group.IsDir() {
			continue
		}
		if strings.HasPrefix(group.Name(), ".") {
			continue
		}
		fanOutGroups <- group.Name()
	}
	close(fanOutGroups)

	<-done
	return nil
}

const (
	updateFile = ".update"
)

// Watch polls the filesystem and kicks off a new domain directory scan when needed.
func Watch(rootDomain string, updater domainsUpdater, interval time.Duration) {
	lastUpdate := []byte("no-update")

	for {
		// Read the update file
		update, err := ioutil.ReadFile(updateFile)
		if err != nil && !os.IsNotExist(err) {
			log.WithError(err).Print("failed to read update timestamp")
			time.Sleep(interval)
			continue
		}

		// If it's the same ignore
		if bytes.Equal(lastUpdate, update) {
			time.Sleep(interval)
			continue
		}
		lastUpdate = update

		started := time.Now()
		dm := make(Map)
		if err := dm.ReadGroups(rootDomain); err != nil {
			log.WithError(err).Warn("domain scan failed")
		}
		duration := time.Since(started).Seconds()

		var hash string
		if len(update) < 1 {
			hash = "<empty>"
		} else {
			hash = strings.TrimSpace(string(update))
		}

		logConfiguredDomains(dm)

		log.WithFields(log.Fields{
			"count(domains)": len(dm),
			"duration":       duration,
			"hash":           hash,
		}).Info("Updated all domains")

		if updater != nil {
			updater(dm)
		}

		// Update prometheus metrics
		metrics.DomainLastUpdateTime.Set(float64(time.Now().UTC().Unix()))
		metrics.DomainsServed.Set(float64(len(dm)))
		metrics.DomainUpdates.Inc()

		time.Sleep(interval)
	}
}

func logConfiguredDomains(dm Map) {
	if log.GetLevel() != log.DebugLevel {
		return
	}

	for h, d := range dm {
		log.WithFields(log.Fields{
			"domain": d,
			"host":   h,
		}).Debug("Configured domain")
	}
}
