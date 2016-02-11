package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type domains map[string]*domain

type domainsUpdater func(domains domains)

func (d domains) addDomain(group, project string, config *domainConfig) error {
	newDomain := &domain{
		Group:   group,
		Project: project,
		Config:  config,
	}

	if config != nil {
		d[config.Domain] = newDomain
	} else {
		domainName := group + "." + *pagesDomain
		d[domainName] = newDomain
	}
	return nil
}

func (d domains) readProjects(group string) (count int) {
	projects, err := os.Open(filepath.Join(*pagesRoot, group))
	if err != nil {
		return
	}
	defer projects.Close()

	fis, err := projects.Readdir(0)
	if err != nil {
		log.Println("Failed to Readdir for ", *pagesRoot, ":", err)
	}

	for _, project := range fis {
		// Ignore non directories
		if !project.IsDir() {
			continue
		}

		// Ignore hidden projects
		if strings.HasPrefix(project.Name(), ".") {
			continue
		}

		// Ignore projects that have .deleted in name
		if strings.HasSuffix(project.Name(), ".deleted") {
			continue
		}

		// Ignore projects without public
		_, err := os.Lstat(filepath.Join(*pagesRoot, group, project.Name(), "public"))
		if err != nil {
			continue
		}

		count++

		var config domainsConfig
		err = config.Read(group, project.Name())
		log.Println(err)
		if err == nil {
			for _, domainConfig := range config.Domains {
				if domainConfig.Valid() {
					d.addDomain(group, project.Name(), &domainConfig)
				}
			}
		}
	}
	return
}

func (d domains) ReadGroups() error {
	groups, err := os.Open(*pagesRoot)
	if err != nil {
		return err
	}
	defer groups.Close()

	fis, err := groups.Readdir(0)
	if err != nil {
		log.Println("Failed to Readdir for ", *pagesRoot, ":", err)
	}

	for _, group := range fis {
		if !group.IsDir() {
			continue
		}
		if strings.HasPrefix(group.Name(), ".") {
			continue
		}

		count := d.readProjects(group.Name())
		if count > 0 {
			d.addDomain(group.Name(), "", nil)
		}
	}
	return nil
}

func watchDomains(updater domainsUpdater, interval time.Duration) {
	lastUpdate := []byte("no-update")

	for {
		update, err := ioutil.ReadFile(filepath.Join(*pagesRoot, ".update"))
		if bytes.Equal(lastUpdate, update) {
			if err != nil {
				log.Println("Failed to read update timestamp:", err)
				time.Sleep(interval)
			}
			continue
		}
		lastUpdate = update

		started := time.Now()
		domains := make(domains)
		domains.ReadGroups()
		duration := time.Since(started)
		log.Println("Updated", len(domains), "domains in", duration)

		if updater != nil {
			updater(domains)
		}
		time.Sleep(interval)
	}
}
