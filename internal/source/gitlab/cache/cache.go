package cache

import (
	"context"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	client Resolver
	store  Store
}

// NewCache creates a new instance of Cache.
func NewCache(client Resolver) *Cache {
	return &Cache{
		client: client,
		store:  newMemStore(),
	}
}

// Resolve is going to return a Lookup based on a domain name
func (c *Cache) Resolve(ctx context.Context, domain string) *Lookup {
	entry := c.store.LoadOrCreate(domain)

	if entry.IsUpToDate() {
		return entry.Lookup()
	}

	if entry.NeedsRefresh() {
		entry.Refresh(c.client, c.store)

		return entry.Lookup()
	}

	return entry.Retrieve(ctx, c.client)
}
