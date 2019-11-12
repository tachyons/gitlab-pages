package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type client struct {
	resolutions uint64
	domain      chan string
	failure     error
	status      int
}

func (c *client) Resolve(ctx context.Context, _ string) Lookup {
	atomic.AddUint64(&c.resolutions, 1)

	if c.status == 0 {
		c.status = 200
	}

	if c.failure != nil {
		return Lookup{Domain: Domain{}, Status: c.status, Err: c.failure}
	}

	return Lookup{Domain: Domain{Name: <-c.domain}, Status: c.status, Err: nil}
}

func withTestCache(config resolverConfig, block func(*Cache, *client)) {
	var resolver *client

	if config.buffered {
		resolver = &client{domain: make(chan string, 1), failure: config.failure}
	} else {
		resolver = &client{domain: make(chan string), failure: config.failure}
	}

	cache := NewCache(resolver)

	block(cache, resolver)
}

func (cache *Cache) withTestEntry(config entryConfig, block func()) {
	domain := "my.gitlab.com"

	if len(config.domain) > 0 {
		domain = config.domain
	}

	entry := cache.store.ReplaceOrCreate(context.Background(), domain)

	if config.retrieved {
		newResponse := make(chan Lookup, 1)
		newResponse <- Lookup{Domain: Domain{Name: domain}, Status: 200}
		entry.setResponse(newResponse)
	}

	if config.expired {
		entry.created = time.Now().Add(-time.Hour)
	}

	block()
}

type resolverConfig struct {
	buffered bool
	failure  error
}

type entryConfig struct {
	domain    string
	expired   bool
	retrieved bool
}

func TestGetLookup(t *testing.T) {
	t.Run("when item is not cached", func(t *testing.T) {
		withTestCache(resolverConfig{buffered: true}, func(cache *Cache, resolver *client) {
			resolver.domain <- "my.gitlab.com"

			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			assert.NoError(t, lookup.Err)
			assert.Equal(t, 200, lookup.Status)
			assert.Equal(t, "my.gitlab.com", lookup.Domain.Name)
			assert.Equal(t, uint64(1), resolver.resolutions)
		})
	})

	t.Run("when item is not cached and accessed multiple times", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			wg := &sync.WaitGroup{}
			ctx := context.Background()

			receiver := func() {
				defer wg.Done()
				cache.Resolve(ctx, "my.gitlab.com")
			}

			wg.Add(3)
			go receiver()
			go receiver()
			go receiver()

			assert.Equal(t, uint64(0), resolver.resolutions)

			resolver.domain <- "my.gitlab.com"
			wg.Wait()

			assert.Equal(t, uint64(1), resolver.resolutions)
		})
	})

	t.Run("when item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: true}, func() {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, "my.gitlab.com", lookup.Domain.Name)
				assert.Equal(t, uint64(0), resolver.resolutions)
			})
		})
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func() {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, "my.gitlab.com", lookup.Domain.Name)
				assert.Equal(t, uint64(0), resolver.resolutions)

				resolver.domain <- "my.gitlab.com"
				assert.Equal(t, uint64(1), resolver.resolutions)
			})
		})
	})

	t.Run("when item in long cache is requested multiple times", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func() {
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, uint64(0), resolver.resolutions)

				resolver.domain <- "my.gitlab.com"
				assert.Equal(t, uint64(1), resolver.resolutions)
			})
		})
	})

	t.Run("when retrieval failed with an error", func(t *testing.T) {
		withTestCache(resolverConfig{failure: errors.New("500 err")}, func(cache *Cache, resolver *client) {
			maxRetrievalInterval = 0

			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			assert.Equal(t, uint64(3), resolver.resolutions)
			assert.EqualError(t, lookup.Err, "500 err")
		})
	})
}
