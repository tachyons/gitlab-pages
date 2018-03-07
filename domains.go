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

func (d domains) addDomain(rootDomain, group, project string, config *domainConfig) error {
	newDomain := &domain{
		Group:   group,
		Project: project,
		Config:  config,
	}

	var domainName string
	if config != nil {
		domainName = config.Domain
	} else {
		domainName = group + "." + rootDomain
	}
	domainName = strings.ToLower(domainName)
	d[domainName] = newDomain

	return nil
}

func (d domains) readProjectConfig(rootDomain, group, project string) (err error) {
	var config domainsConfig
	err = config.Read(group, project)
	if err != nil {
		return
	}

	for _, domainConfig := range config.Domains {
		config := domainConfig // domainConfig is reused for each loop iteration
		if domainConfig.Valid(rootDomain) {
			d.addDomain(rootDomain, group, project, &config)
		}
	}
	return
}

func (d domains) readProject(rootDomain, group, project string) error {
	if strings.HasPrefix(project, ".") {
		return errors.New("hidden project")
	}

	// Ignore projects that have .deleted in name
	if strings.HasSuffix(project, ".deleted") {
		return errors.New("deleted project")
	}

	_, err := os.Lstat(filepath.Join(group, project, "public"))
	if err != nil {
		return errors.New("missing public/ in project")
	}

	d.readProjectConfig(rootDomain, group, project)
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

		count := d.readProjects(rootDomain, group.Name())
		if count > 0 {
			d.addDomain(rootDomain, group.Name(), "", nil)
		}
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
