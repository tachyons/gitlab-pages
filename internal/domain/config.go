package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config represents a custom domain config
type Config struct {
	Domain        string
	Certificate   string
	Key           string
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// MultiConfig represents a group of custom domain configs
type MultiConfig struct {
	Domains       []Config
	HTTPSOnly     bool   `json:"https_only"`
	ID            uint64 `json:"id"`
	AccessControl bool   `json:"access_control"`
}

// Valid validates a custom domain config for a root domain
func (c *Config) Valid(rootDomain string) bool {
	if c.Domain == "" {
		return false
	}

	// TODO: better sanitize domain
	domain := strings.ToLower(c.Domain)
	rootDomain = "." + rootDomain
	return !strings.HasSuffix(domain, rootDomain)
}

func (c *MultiConfig) Read(group, project string) (err error) {
	configFile, err := os.Open(filepath.Join(group, project, "config.json"))
	if err != nil {
		return err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(c)
	return
}
