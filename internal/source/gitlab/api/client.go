package api

import (
	"context"
)

// Client represents an interface we use to retrieve information from GitLab
type Client interface {
	// Resolve retrieves an VirtualDomain from the GitLab API and wraps it into a Lookup
	GetLookup(ctx context.Context, domain string) Lookup
}
