package gitlab

import "context"

// Client is an internal HTTP client used for communication with GitLab
// instance
type Client interface {
	Resolve(ctx context.Context, domain string) (*Lookup, int, error)
}
