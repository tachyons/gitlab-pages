package cache

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/labkit/correlation"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	store     Store
	retriever *Retriever
}

// NewCache creates a new instance of Cache.
func NewCache(client api.Client, cc *config.Cache) *Cache {
	r := NewRetriever(client, cc.RetrievalTimeout, cc.MaxRetrievalInterval, cc.MaxRetrievalRetries)
	return &Cache{
		store:     newMemStore(cc),
		retriever: r,
	}
}

// Resolve is going to return a lookup based on a domain name. The caching
// algorithm works as follows:
// - We first check if the cache entry exists, and if it is up-to-date. If it
//   is fresh we return the lookup entry from cache and it is a cache hit.
// - If entry is not up-to-date, that means it has been created in a cache
//   more than `entryRefreshTimeout` duration ago,  we schedule an asynchronous
//   retrieval of the latest configuration we are going to obtain through the
//   API, and we immediately return an old value, to avoid blocking clients. In
//   this case it is also a cache hit.
// - If cache entry has not been populated with a lookup information yet, we
//   block all the clients and make them wait until we retrieve the lookup from
//   the GitLab API. Clients should not wait for longer than
//   `retrievalTimeout`. It is a cache miss.
//
// We are going to retrieve a lookup from GitLab API using a retriever type. In
// case of failures (when GitLab API client returns an error) we will retry the
// operation a few times, waiting `maxRetrievalInterval` in between, total
// amount of requests is defined as `maxRetrievalRetries`. In case of an
// erroneous response, we will cache it, and it get recycled as every other
// cache entry.
//
// Examples:
// 1. Everything works
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - we successfully retrieve a lookup
//  - we cache this response
//  - and we pass it upstream to all clients
// 2. A domain does not exist
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - GitLab responded with a lookup and 204 HTTP status
//  - we cache this response with domain being `nil`
//  - we pass this lookup upstream to all the clients
// 3. GitLab is not responding
//  - a client opens pages
//  - we create a new cache entry
//  - cache entry needs a warm up
//  - a client waits until we retrieve a lookup
//  - GitLab does not respond or responds with an error
//  - we retry this retrieval every `maxRetrievalInterval`
//  - we retry this retrieval `maxRetrievalRetries` in total
//  - we create a lookup that contains information about an error
//  - we cache this response
//  - we pass this lookup upstream to all the clients
func (c *Cache) Resolve(ctx context.Context, domain string) *api.Lookup {
	entry := c.store.LoadOrCreate(domain)

	if entry.IsUpToDate() {
		metrics.DomainsSourceCacheHit.Inc()
		return entry.Lookup()
	}

	if entry.NeedsRefresh() {
		c.Refresh(entry)

		metrics.DomainsSourceCacheHit.Inc()
		return entry.Lookup()
	}

	metrics.DomainsSourceCacheMiss.Inc()
	return c.retrieve(ctx, entry)
}

func (c *Cache) retrieve(ctx context.Context, entry *Entry) *api.Lookup {
	// We run the code within an additional func() to run both `e.setResponse`
	// and `c.retriever.Retrieve` asynchronously.
	// We are using a sync.Once so this assumes that setResponse is always called
	// the first (and only) time f is called, otherwise future requests will hang.
	entry.retrieve.Do(func() {
		correlationID := correlation.ExtractFromContext(ctx)

		go func() {
			l := c.retriever.Retrieve(correlationID, entry.domain)
			entry.setResponse(l)
		}()
	})

	var lookup *api.Lookup
	select {
	case <-ctx.Done():
		lookup = &api.Lookup{Name: entry.domain, Error: fmt.Errorf("original context done: %w", ctx.Err())}
	case <-entry.retrieved:
		lookup = entry.Lookup()
	}

	return lookup
}

// Refresh will update the entry in the store only when it gets resolved successfully.
// If an existing successful entry exists, it will only be replaced if the new resolved
// entry is successful too.
// Errored refreshed Entry responses will not replace the previously successful entry.response
// for a maximum time of e.expirationTimeout.
func (c *Cache) Refresh(entry *Entry) {
	entry.refresh.Do(func() {
		go c.refreshFunc(entry)
	})
}

func (c *Cache) refreshFunc(e *Entry) {
	entry := newCacheEntry(e.domain, e.refreshTimeout, e.expirationTimeout)

	c.retrieve(context.Background(), entry)

	// do not replace existing Entry `e.response` when `entry.response` has an error
	// and `e` has not expired. See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/281.
	if !e.isExpired() && entry.hasTemporaryError() {
		entry.response = e.response
		entry.refreshedOriginalTimestamp = e.created
	}

	c.store.ReplaceOrCreate(e.domain, entry)
}
