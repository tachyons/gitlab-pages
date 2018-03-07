package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

type domains map[string]*domain

type domainsUpdater func(domains domains)

func (d domains) addDomain(rootDomain, group, projectName string, config *domainConfig) error {
	newDomain := &domain{
		Group:       group,
		ProjectName: projectName,
		Config:      config,
	}

	var domainName string
	domainName = strings.ToLower(config.Domain)
	d[domainName] = newDomain

	return nil
}

func (d domains) updateGroupDomain(rootDomain, group, projectName string, httpsOnly bool) error {
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

	return nil
}

func (d domains) readProjectConfig(rootDomain, group, projectName string) (err error) {
	var config domainsConfig
	err = config.Read(group, projectName)
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
	return
}

func (d domains) readProject(rootDomain, group, projectName string) error {
	if strings.HasPrefix(projectName, ".") {
		return errors.New("hidden project")
	}

	// Ignore projects that have .deleted in name
	if strings.HasSuffix(projectName, ".deleted") {
		return errors.New("deleted project")
	}

	_, err := os.Lstat(filepath.Join(group, projectName, "public"))
	if err != nil {
		return errors.New("missing public/ in project")
	}

	d.readProjectConfig(rootDomain, group, projectName)
	return nil
}

func (d domains) readProjects(rootDomain, group string) (count int) {
	projects, err := os.Open(group)
	if err != nil {
		return
	}
	defer projects.Close()

	fis, err := projects.Readdir(0)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"group": group,
		}).Print("readdir failed")
	}

	for _, project := range fis {
		// Ignore non directories
		if !project.IsDir() {
			continue
		}

		err := d.readProject(rootDomain, group, project.Name())
		if err == nil {
			count++
		}
	}
	return
}

func (d domains) ReadGroups(rootDomain string) error {
	groups, err := os.Open(".")
	if err != nil {
		return err
	}
	defer groups.Close()

	fis, err := groups.Readdir(0)
	if err != nil {
		log.WithError(err).Print("readdir failed")
	}

	for _, group := range fis {
		if !group.IsDir() {
			continue
		}
		if strings.HasPrefix(group.Name(), ".") {
			continue
		}

		d.readProjects(rootDomain, group.Name())
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
		domains.ReadGroups(rootDomain)
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
