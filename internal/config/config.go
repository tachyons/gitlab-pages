package config

import (
	"time"
)

// Default configuration that can be accessed by different packages
var Default = &Config{
	Zip: &ZipServing{},
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
