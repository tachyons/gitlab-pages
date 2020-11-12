package config

import "time"

// TODO: refactor config flags in main.go and find a better way to handle all settings

// ZipVFSConfig struct that can be accessed by different packages to share
// configuration parameters
var ZipVFSConfig *ZipServing

// ZipServing stores all configuration values to be used by the zip VFS opening and
// caching
type ZipServing struct {
	ExpirationInterval           time.Duration
	CleanupInterval              time.Duration
	RefreshInterval              time.Duration
	OpenTimeout                  time.Duration
	DataOffsetItems              int64
	DataOffsetExpirationInterval time.Duration
	ReadlinkItems                int64
	ReadlinkExpirationInterval   time.Duration
}
