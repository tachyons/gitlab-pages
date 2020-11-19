package config

import (
	"time"

	"github.com/namsral/flag"
)

// Default configuration that can be accessed by different packages
var Default *Config

// TODO: move all flags to this package, including flag.Parse()
func init() {
	Default = &Config{
		Zip: &ZipServing{},
	}

	flag.DurationVar(&Default.Zip.ExpirationInterval, "zip-cache-expiration", 60*time.Second, "Zip serving archive cache expiration interval")
	flag.DurationVar(&Default.Zip.CleanupInterval, "zip-cache-cleanup", 30*time.Second, "Zip serving archive cache cleanup interval")
	flag.DurationVar(&Default.Zip.RefreshInterval, "zip-cache-refresh", 30*time.Second, "Zip serving archive cache refresh interval")
	flag.DurationVar(&Default.Zip.OpenTimeout, "zip-open-timeout", 30*time.Second, "Zip archive open timeout")
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
