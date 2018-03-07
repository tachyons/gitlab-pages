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

func (c *domainConfig) Valid(rootDomain string) bool {
	if c.Domain == "" {
		return false
	}

	// TODO: better sanitize domain
	domain := strings.ToLower(c.Domain)
	rootDomain = "." + rootDomain
	return !strings.HasSuffix(domain, rootDomain)
}

func (c *domainsConfig) Read(group, project string) (err error) {
	configFile, err := os.Open(filepath.Join(domainRoot, group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(c)
	return
}
