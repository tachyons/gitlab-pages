package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type domains map[string]domain

type domainsUpdater func(domains domains)

func isDomainAllowed(domain string) bool {
	if domain == "" {
		return false
	}
	// TODO: better sanitize domain
	domain = strings.ToLower(domain)
	pagesDomain = "." + strings.ToLower(pagesDomain)
	return !strings.HasPrefix(domain, pagesDomain)
}

func (d domains) addDomain(group, project string, config *domainConfig) error {
	newDomain := &domain{
		Group:   group,
		Project: project,
		Config:  domainConfig,
	}

	if config != nil {
		if !isDomainAllowed(domainConfig.Domain) {
			return errors.New("domain name is not allowed")
		}

		d[config.Domain] = newDomain
	} else {
		domainName := group + "." + *pagesDomain
		d[domainName] = newDomain
	}
	return
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
		if !project.IsDir() {
			continue
		}
		if strings.HasPrefix(project.Name(), ".") {
			continue
		}

		count++

		var config domainsConfig
		err := config.Read(group, project.Name())
		if err != nil {
			continue
		}

		for _, domainConfig := range domainsConfig.Domains {
			d.addDomain(group, project.Name(), &domainConfig)
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
			d.addDomain(group, "", &domainConfig)
		}
	}
	return nil
}

func watchDomains(updater domainsUpdater) {
	lastUpdate := "no-configuration"

	for {
		update, err := ioutil.ReadFile(filepath.Join(*pagesRoot, ".update"))
		if bytes.Equal(lastUpdate, update) {
			if err != nil {
				log.Println("Failed to read update timestamp:", err)
				time.Sleep(time.Second)
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
	}
}
