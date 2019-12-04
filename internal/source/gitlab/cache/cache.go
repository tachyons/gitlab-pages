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

// Resolve is going to return a Lookup based on a domain name. The caching
// algorithm works as follows:
// - We first check if the cache entry exists, and if it is up-to-date. If it
//   is fresh we return the &Lookup entry from cache and it is a cache hit.
// - If entry is not up-to-date, we schedule an asynchronous retrieval of the
//   latest configuration we are going to obtain through the API, and we
//   immediately return an old value, to avoid blocking clients. In this case
//   it is also a cache hit.
// - If cache entry has not been populated with a Lookup information yet, we
//   block all the clients and make them wait until we retrieve the Lookup from
//   the GitLab API.
//
// We are going to retrieve a Lookup from GitLab API using Retriever type. In
// case of failures (when GitLab API client returns an error) we will retry the
// operation a few times. In case of an erroneous response, we will cache it,
// and it get recycled as every other cache entry.
//
// Examples:
// 1. Everything works
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs warm up
//  - a client waits until we retrieve a lookup
//  - we successfuly retrieve a lookup
//  - we cache this response
//  - and we pass it upstream to all clients
// 2. A domain does not exist
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs warm up
//  - a client waits until we retrieve a lookup
//  - GitLab responded with a lookup that contains information about domain
//    not being found
//  - we cache this response
//  - we pass this lookup upstream to all theclient
// 3. GitLab is not responding
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs warm up
//  - a client waits until we retrieve a lookup
//  - GitLab does not respond or responds with an error
//  - we retry this information a few times
//  - we create a lookup that contains information about an error
//  - we cache this response
//  - we pass this lookup upstream to all the clients
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
