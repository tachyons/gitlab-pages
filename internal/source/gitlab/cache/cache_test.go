package cache

import (
	"context"
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

func (s *client) Resolve(ctx context.Context, domain string) (*Lookup, int, error) {
	atomic.AddUint64(&s.resolutions, 1)

	if s.failure != nil {
		return &Lookup{}, s.status, s.failure
	}

	return &Lookup{Domain: <-s.domain}, 200, nil
}

func withTestCache(config resolverConfig, block func(*Cache, *client)) {
	var resolver *client

	if config.buffered {
		resolver = &client{domain: make(chan string, 1)}
	} else {
		resolver = &client{domain: make(chan string)}
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
		newResponse := make(chan Response, 1)
		newResponse <- Response{lookup: &Lookup{Domain: domain}, status: 200}
		entry.setResponse(newResponse)
	}

	if config.expired {
		entry.created = time.Now().Add(-time.Hour)
	}

	block()
}

type resolverConfig struct {
	buffered bool
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

			lookup, status, err := cache.Resolve(context.Background(), "my.gitlab.com")

			assert.NoError(t, err)
			assert.Equal(t, 200, status)
			assert.Equal(t, "my.gitlab.com", lookup.Domain)
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
				lookup, _, _ := cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, "my.gitlab.com", lookup.Domain)
				assert.Equal(t, uint64(0), resolver.resolutions)
			})
		})
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func() {
				lookup, _, _ := cache.Resolve(context.Background(), "my.gitlab.com")

				assert.Equal(t, "my.gitlab.com", lookup.Domain)
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
		// cache := NewCache()
		// resolver := &stubbedClient{
		// 	failure: errors.New("could not retrieve lookup"),
		// }
		//
		// lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)
		//
		// assert.Equal(t, &Lookup{}, lookup)
	})
}
