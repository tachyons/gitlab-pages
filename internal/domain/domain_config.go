package domain

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
	HTTPSOnly   bool `json:"https_only"`
}

type domainsConfig struct {
	Domains       []domainConfig
	HTTPSOnly     bool   `json:"https_only"`
	Private       bool   `json:"private"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
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
	configFile, err := os.Open(filepath.Join(group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(c)
	return
}
