package config

import (
	"time"
)

// TODO: refactor config flags in main.go and find a better way to handle all settings

type Config struct {
	zip *ZipServing
}

// DefaultConfig struct that can be accessed by different packages to share
// configuration parameters
var DefaultConfig = &Config{
	// TODO: think of a way to not repeat these here and in main.go
	zip: &ZipServing{
		ExpirationInterval: time.Minute,
		CleanupInterval:    time.Minute / 2,
		RefreshInterval:    time.Minute / 2,
		OpenTimeout:        time.Minute / 2,
	},
}

// ZipServing stores all configuration values to be used by the zip VFS opening and
// caching
type ZipServing struct {
	ExpirationInterval time.Duration
	CleanupInterval    time.Duration
	RefreshInterval    time.Duration
	OpenTimeout        time.Duration
}

// SetZip config to the global config
func (c *Config) SetZip(zip *ZipServing) {
	c.zip = zip
}

func (c *Config) GetZip() *ZipServing {
	return c.zip
}
