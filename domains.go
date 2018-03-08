package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

func (d domains) readProjectConfig(rootDomain, group, projectName string) {
	var config domainsConfig
	err := config.Read(group, projectName)
	if err != nil {
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

func (d domains) readProject(rootDomain, group, projectName string) {
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

	d.readProjectConfig(rootDomain, group, projectName)
}

func (d domains) readProjects(rootDomain, group string, buf []byte) {
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

		d.readProject(rootDomain, group, project.Name())
	}
}

func (d domains) ReadGroups(rootDomain string) error {
	buf := make([]byte, 2*os.Getpagesize())

	fis, err := godirwalk.ReadDirents(".", buf)
	if err != nil {
		return err
	}

	for _, group := range fis {
		if !group.IsDir() {
			continue
		}
		if strings.HasPrefix(group.Name(), ".") {
			continue
		}

		d.readProjects(rootDomain, group.Name(), buf)
	}

	return nil
}

func watchDomains(rootDomain string, updater domainsUpdater, interval time.Duration) {
	lastUpdate := []byte("no-update")

	for {
		// Read the update file
		update, err := ioutil.ReadFile(".update")
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
