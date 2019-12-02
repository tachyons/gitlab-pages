package cache

import (
	"context"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	client api.Client
	store  Store
}

// NewCache creates a new instance of Cache.
func NewCache(client api.Client) *Cache {
	return &Cache{
		client: client,
		store:  newMemStore(),
	}
}

// Resolve is going to return a Lookup based on a domain name
func (c *Cache) Resolve(ctx context.Context, domain string) *api.Lookup {
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

// New creates a new instance of Cache and sets default expiration
func New() *Cache {
	return &Cache{}
}
