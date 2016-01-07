package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type domains map[string]domain

type domainsUpdater func(domains domains)

func readGroups(domains domains) error {
	groups, err := filepath.Glob(filepath.Join(*pagesRoot, "*/"))
	if err != nil {
		return err
	}

	for _, groupDir := range groups {
		group := filepath.Base(groupDir)
		groupName := strings.ToLower(group)
		domains[groupName+"."+*pagesDomain] = domain{
			Group: group,
			CNAME: false,
		}
	}
	return nil
}

func readCnames(domains domains) error {
	cnames, err := filepath.Glob(filepath.Join(*pagesRoot, "*/*/CNAME"))
	if err != nil {
		return err
	}

	for _, cnamePath := range cnames {
		cnameData, err := ioutil.ReadFile(cnamePath)
		if err != nil {
			continue
		}

		for _, cname := range strings.Fields(string(cnameData)) {
			cname := strings.ToLower(cname)
			if strings.HasSuffix(cname, "."+*pagesDomain) {
				continue
			}

			domains[cname] = domain{
				// TODO: make it nicer
				Group:   filepath.Base(filepath.Dir(filepath.Dir(cnamePath))),
				Project: filepath.Base(filepath.Dir(cnamePath)),
				CNAME:   true,
			}
		}
	}
	return nil
}

func watchDomains(updater domainsUpdater) {
	var lastModified time.Time

	for {
		fi, err := os.Stat(*pagesRoot)
		if err != nil || !fi.IsDir() {
			log.Println("Failed to read domains from", *pagesRoot, "due to:", err, fi.IsDir())
			time.Sleep(time.Second)
			continue
		}

		// If directory did not get modified we will reload
		if !lastModified.Before(fi.ModTime()) {
			time.Sleep(time.Second)
			continue
		}
		lastModified = fi.ModTime()

		started := time.Now()
		domains := make(domains)
		readGroups(domains)
		readCnames(domains)
		duration := time.Since(started)
		log.Println("Updated", len(domains), "domains in", duration)
		if updater != nil {
			updater(domains)
		}
	}
}
