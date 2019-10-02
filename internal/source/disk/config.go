package disk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DomainConfig represents a custom domain config
type domainConfig struct {
	Domain        string
	Certificate   string
	Key           string
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// MultiDomainConfig represents a group of custom domain configs
type multiDomainConfig struct {
	Domains       []domainConfig
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// ProjectConfig is a project-level configuration
type projectConfig struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}

// Valid validates a custom domain config for a root domain
func (c *domainConfig) Valid(rootDomain string) bool {
	if c.Domain == "" {
		return false
	}

	// TODO: better sanitize domain
	domain := strings.ToLower(c.Domain)
	rootDomain = "." + rootDomain
	return !strings.HasSuffix(domain, rootDomain)
}

// Read reads a multi domain config and decodes it from a `config.json`
func (c *multiDomainConfig) Read(group, project string) error {
	configFile, err := os.Open(filepath.Join(group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	return json.NewDecoder(configFile).Decode(c)
}
