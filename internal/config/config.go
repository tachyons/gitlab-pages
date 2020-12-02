package config

import (
	"time"
)

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
