package dirs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DomainConfig represents a custom domain config
type DomainConfig struct {
	Domain        string
	Certificate   string
	Key           string
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// MultiDomainConfig represents a group of custom domain configs
type MultiDomainConfig struct {
	Domains       []DomainConfig
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// ProjectConfig is a project-level configuration
type ProjectConfig struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}

// Valid validates a custom domain config for a root domain
func (c *DomainConfig) Valid(rootDomain string) bool {
	if c.Domain == "" {
		return false
	}

	// TODO: better sanitize domain
	domain := strings.ToLower(c.Domain)
	rootDomain = "." + rootDomain
	return !strings.HasSuffix(domain, rootDomain)
}

// Read reads a multi domain config and decodes it from a `config.json`
func (c *MultiDomainConfig) Read(group, project string) error {
	configFile, err := os.Open(filepath.Join(group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	return json.NewDecoder(configFile).Decode(c)
}
