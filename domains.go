package main

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

type domains map[string]*domain

type domainsUpdater func(domains domains)

func (d domains) addDomain(rootDomain, group, projectName string, config *domainConfig) {
	newDomain := &domain{
		Group:       group,
		ProjectName: projectName,
		Config:      config,
	}

	var domainName string
	domainName = strings.ToLower(config.Domain)
	d[domainName] = newDomain
}

func (d domains) updateGroupDomain(rootDomain, group, projectName string, httpsOnly bool) {
	domainName := strings.ToLower(group + "." + rootDomain)
	groupDomain := d[domainName]

	if groupDomain == nil {
		groupDomain = &domain{
			Group:    group,
			Projects: make(projects),
		}
	}

	groupDomain.Projects[projectName] = &project{
		HTTPSOnly: httpsOnly,
	}
	d[domainName] = groupDomain
}

func (d domains) readProjectConfig(rootDomain string, group, projectName string, config *domainsConfig) {
	if config == nil {
		// This is necessary to preserve the previous behaviour where a
		// group domain is created even if no config.json files are
		// loaded successfully. Is it safe to remove this?
		d.updateGroupDomain(rootDomain, group, projectName, false)
		return
	}

	d.updateGroupDomain(rootDomain, group, projectName, config.HTTPSOnly)

	for _, domainConfig := range config.Domains {
		config := domainConfig // domainConfig is reused for each loop iteration
		if domainConfig.Valid(rootDomain) {
			d.addDomain(rootDomain, group, projectName, &config)
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

func (d domains) ReadGroups(rootDomain string) error {
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
				readProjects(group, buf, fanIn)
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
			d.readProjectConfig(rootDomain, result.group, result.project, result.config)
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

func watchDomains(rootDomain string, updater domainsUpdater, interval time.Duration) {
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
		domains := make(domains)
		if err := domains.ReadGroups(rootDomain); err != nil {
			log.WithError(err).Warn("domain scan failed")
		}
		duration := time.Since(started).Seconds()

		log.WithFields(log.Fields{
			"domains":  len(domains),
			"duration": duration,
			"hash":     update,
		}).Print("updated domains")

		if updater != nil {
			updater(domains)
		}

		// Update prometheus metrics
		metrics.DomainLastUpdateTime.Set(float64(time.Now().UTC().Unix()))
		metrics.DomainsServed.Set(float64(len(domains)))
		metrics.DomainUpdates.Inc()

		time.Sleep(interval)
	}
}
