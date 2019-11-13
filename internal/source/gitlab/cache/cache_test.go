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
	started     uint64
	resolutions uint64
	bootup      chan uint64
	domain      chan string
	failure     error
	status      int
}

func (c *client) Resolve(ctx context.Context, _ string) Lookup {
	var domain Domain

	c.bootup <- atomic.AddUint64(&c.started, 1)
	defer atomic.AddUint64(&c.resolutions, 1)

	if c.status == 0 {
		c.status = 200
	}

	if c.failure == nil {
		domain = Domain{Name: <-c.domain}
	}

	return Lookup{Domain: domain, Status: c.status, Error: c.failure}
}

func withTestCache(config resolverConfig, block func(*Cache, *client)) {
	var chanSize int

	if config.buffered {
		chanSize = 1
	} else {
		chanSize = 0
	}

	resolver := &client{
		domain:  make(chan string, chanSize),
		bootup:  make(chan uint64, 100),
		failure: config.failure,
	}

	cache := NewCache(resolver)

	block(cache, resolver)
}

func (cache *Cache) withTestEntry(config entryConfig, block func(*Entry)) {
	domain := "my.gitlab.com"

	if len(config.domain) > 0 {
		domain = config.domain
	}

	entry := cache.store.LoadOrCreate(context.Background(), domain)

	if config.retrieved {
		newResponse := make(chan Lookup, 1)
		newResponse <- Lookup{Domain: Domain{Name: domain}, Status: 200}
		entry.setResponse(newResponse)
	}

	if config.expired {
		entry.created = time.Now().Add(-time.Hour)
	}

	block(entry)
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

func TestResolve(t *testing.T) {
	t.Run("when item is not cached", func(t *testing.T) {
		withTestCache(resolverConfig{buffered: true}, func(cache *Cache, resolver *client) {
			resolver.domain <- "my.gitlab.com"

			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			assert.NoError(t, lookup.Error)
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
			cache.withTestEntry(entryConfig{expired: false, retrieved: true}, func(*Entry) {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, "my.gitlab.com", lookup.Domain.Name)
				assert.Equal(t, uint64(0), resolver.resolutions)
			})
		})
	})

	t.Run("when a non-retrieved new item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(*Entry) {
				lookup := make(chan *Lookup, 1)

				go func() {
					lookup <- cache.Resolve(context.Background(), "my.gitlab.com")
				}()

				<-resolver.bootup

				assert.Equal(t, uint64(1), resolver.started)
				assert.Equal(t, uint64(0), resolver.resolutions)

				resolver.domain <- "my.gitlab.com"
				<-lookup

				assert.Equal(t, uint64(1), resolver.started)
				assert.Equal(t, uint64(1), resolver.resolutions)
			})
		})
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
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
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
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
			assert.EqualError(t, lookup.Error, "500 err")
		})
	})

	t.Run("when retrieval failed because of an external context timeout", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
		defer cancel()

		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			lookup := cache.Resolve(ctx, "my.gitlab.com")

			assert.Equal(t, uint64(0), resolver.resolutions)
			assert.EqualError(t, lookup.Error, "context timeout")
		})
	})

	t.Run("when retrieval failed because of an internal context timeout", func(t *testing.T) {
		retrievalTimeout = 0
		defer func() { retrievalTimeout = 5 * time.Second }()

		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			assert.Equal(t, uint64(0), resolver.resolutions)
			assert.EqualError(t, lookup.Error, "context timeout")
		})
	})

	t.Run("when cache entry is evicted from cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(entry *Entry) {
				ctx := context.Background()
				lookup := make(chan *Lookup, 1)
				go func() { lookup <- cache.Resolve(ctx, "my.gitlab.com") }()

				cache.store.ReplaceOrCreate(ctx, "my.gitlab.com", newCacheEntry(ctx, "my.gitlab.com"))

				resolver.domain <- "my.gitlab.com"
				<-lookup

				assert.EqualError(t, entry.ctx.Err(), "context canceled")
			})
		})
	})
}
