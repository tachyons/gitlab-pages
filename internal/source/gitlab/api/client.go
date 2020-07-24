package api

import (
	"context"
)

// Client represents an interface we use to retrieve information from GitLab
type Client interface {
	// GetLookup retrieves a VirtualDomain from the GitLab API and wraps it into Lookup
	GetLookup(ctx context.Context, domain string) Lookup
}
