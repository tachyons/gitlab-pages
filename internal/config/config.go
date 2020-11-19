package config

import (
	"time"
)

// Default configuration that can be accessed by different packages
var Default = &Config{
	// TODO: remove duplication once all flags are defined in this package
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/507
	Zip: &ZipServing{
		ExpirationInterval: time.Minute,
		CleanupInterval:    time.Minute / 2,
		RefreshInterval:    time.Minute / 2,
		OpenTimeout:        time.Minute / 2,
	},
}

type Config struct {
	Zip *ZipServing
}

// ZipServing stores all configuration values to be used by the zip VFS opening and
// caching
type ZipServing struct {
	ExpirationInterval time.Duration
	CleanupInterval    time.Duration
	RefreshInterval    time.Duration
	OpenTimeout        time.Duration
}
