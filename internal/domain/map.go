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
type Map map[string]*Domain

type domainsUpdater func(Map)

func (dm Map) updateDomainMap(domainName string, domain *Domain) {
	if old, ok := dm[domainName]; ok {
		log.WithFields(log.Fields{
			"domain_name":      domainName,
			"new_group":        domain.group,
			"new_project_name": domain.projectName,
			"old_group":        old.group,
			"old_project_name": old.projectName,
		}).Error("Duplicate domain")
	}

	dm[domainName] = domain
}

func (dm Map) addDomain(rootDomain, groupName, projectName string, config *domainConfig) {
	newDomain := &Domain{
		group:       group{name: groupName},
		projectName: projectName,
		config:      config,
	}

	var domainName string
	domainName = strings.ToLower(config.Domain)
	dm.updateDomainMap(domainName, newDomain)
}

func (dm Map) updateGroupDomain(rootDomain, groupName, projectPath string, httpsOnly bool, accessControl bool, id uint64) {
	domainName := strings.ToLower(groupName + "." + rootDomain)
	groupDomain := dm[domainName]

	if groupDomain == nil {
		groupDomain = &Domain{
			group: group{
				name:      groupName,
				projects:  make(projects),
				subgroups: make(subgroups),
			},
		}
	}

	split := strings.SplitN(strings.ToLower(projectPath), "/", maxProjectDepth)
	projectName := split[len(split)-1]
	g := &groupDomain.group

	for i := 0; i < len(split)-1; i++ {
		subgroupName := split[i]
		subgroup := g.subgroups[subgroupName]
		if subgroup == nil {
			subgroup = &group{
				name:      subgroupName,
				projects:  make(projects),
				subgroups: make(subgroups),
			}
			g.subgroups[subgroupName] = subgroup
		}

		g = subgroup
	}

	g.projects[projectName] = &project{
		NamespaceProject: domainName == projectName,
		HTTPSOnly:        httpsOnly,
		AccessControl:    accessControl,
		ID:               id,
	}

	dm[domainName] = groupDomain
}

func (dm Map) readProjectConfig(rootDomain string, group, projectName string, config *domainsConfig) {
	if config == nil {
		// This is necessary to preserve the previous behaviour where a
		// group domain is created even if no config.json files are
		// loaded successfully. Is it safe to remove this?
		dm.updateGroupDomain(rootDomain, group, projectName, false, false, 0)
		return
	}

	dm.updateGroupDomain(rootDomain, group, projectName, config.HTTPSOnly, config.AccessControl, config.ID)

	for _, domainConfig := range config.Domains {
		config := domainConfig // domainConfig is reused for each loop iteration
		if domainConfig.Valid(rootDomain) {
			dm.addDomain(rootDomain, group, projectName, &config)
		}
	}
}

func readProject(group, parent, projectName string, level int, fanIn chan<- jobResult) {
	if strings.HasPrefix(projectName, ".") {
		return
	}

	// Ignore projects that have .deleted in name
	if strings.HasSuffix(projectName, ".deleted") {
		return
	}

	projectPath := filepath.Join(parent, projectName)
	if _, err := os.Lstat(filepath.Join(group, projectPath, "public")); err != nil {
		// maybe it's a subgroup
		if level <= subgroupScanLimit {
			buf := make([]byte, 2*os.Getpagesize())
			readProjects(group, projectPath, level+1, buf, fanIn)
		}

		return
	}

	// We read the config.json file _before_ fanning in, because it does disk
	// IO and it does not need access to the domains map.
	config := &domainsConfig{}
	if err := config.Read(group, projectPath); err != nil {
		config = nil
	}

	fanIn <- jobResult{group: group, project: projectPath, config: config}
}

func readProjects(group, parent string, level int, buf []byte, fanIn chan<- jobResult) {
	subgroup := filepath.Join(group, parent)
	fis, err := godirwalk.ReadDirents(subgroup, buf)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"group":  group,
			"parent": parent,
		}).Print("readdir failed")
		return
	}

	for _, project := range fis {
		// Ignore non directories
		if !project.IsDir() {
			continue
		}

		readProject(group, parent, project.Name(), level, fanIn)
	}
}

type jobResult struct {
	group   string
	project string
	config  *domainsConfig
}

// ReadGroups walks the pages directory and populates dm with all the domains it finds.
func (dm Map) ReadGroups(rootDomain string, fis godirwalk.Dirents) {
	fanOutGroups := make(chan string)
	fanIn := make(chan jobResult)
	wg := &sync.WaitGroup{}
	for i := 0; i < 4; i++ {
		wg.Add(1)

		go func() {
			buf := make([]byte, 2*os.Getpagesize())

			for group := range fanOutGroups {
				started := time.Now()

				readProjects(group, "", 0, buf, fanIn)

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

		fis, err := godirwalk.ReadDirents(".", nil)
		if err != nil {
			log.WithError(err).Warn("domain scan failed")
			metrics.FailedDomainUpdates.Inc()
			continue
		}

		dm.ReadGroups(rootDomain, fis)
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
