package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type domainConfig struct {
	Domain      string
	Certificate string
	Key         string
}

type domainsConfig struct {
	Domains []domainConfig
}

func (c *domainsConfig) Read(group, project string) (err error) {
	configFile, err := os.Open(filepath.Join(*pagesRoot, project, group, "config.json"))
	if err != nil {
		return nil
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(c)
	return
}
