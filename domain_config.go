package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type domainConfig struct {
	Domain      string
	Certificate string
	Key         string
}

type domainsConfig struct {
	Domains []domainConfig
}

func (c *domainConfig) Valid() bool {
	if c.Domain == "" {
		return false
	}

	// TODO: better sanitize domain
	domain := strings.ToLower(c.Domain)
	rootDomain := "." + strings.ToLower(*pagesDomain)
	return !strings.HasSuffix(domain, rootDomain)
}

func (c *domainsConfig) Read(group, project string) (err error) {
	configFile, err := os.Open(filepath.Join(group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(c)
	return
}
