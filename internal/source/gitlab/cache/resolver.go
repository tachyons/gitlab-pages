package cache

import "context"

// Resolver represents an interface we use retrieve information from cache
type Resolver interface {
	Resolve(ctx context.Context, domain string) (*Lookup, int, error)
}
