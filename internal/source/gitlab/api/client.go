package api

import (
	"context"
	"time"
)

// Client represents an interface we use to retrieve information from GitLab
type Client interface {
	// GetLookup retrives an VirtualDomain from GitLab API and wraps it into Lookup
	GetLookup(ctx context.Context, domain string) Lookup

	Poll(retries int, interval time.Duration, errCh chan error)
}
