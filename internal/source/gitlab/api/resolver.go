package api

import (
	"context"
	"time"
)

// Resolver represents an interface we use to retrieve information from GitLab
// in a more generic way. It can be a concrete API client or cached client.
type Resolver interface {
	// Resolve retrieves an VirtualDomain from GitLab API and wraps it into Lookup
	Resolve(ctx context.Context, domain string) *Lookup
	// Poll test
	Poll(retries int, interval time.Duration, errCh chan error)
}
