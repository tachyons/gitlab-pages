package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type stats struct {
	m       sync.Mutex
	started uint64
	lookups uint64
}

type client struct {
	stats   stats
	bootup  chan uint64
	domain  chan string
	failure error
}

func (s *stats) bumpStarted() uint64 {
	s.m.Lock()
	defer s.m.Unlock()

	s.started++
	return s.started
}

func (s *stats) bumpLookups() uint64 {
	s.m.Lock()
	defer s.m.Unlock()

	s.lookups++
	return s.lookups
}

func (s *stats) getStarted() uint64 {
	s.m.Lock()
	defer s.m.Unlock()

	return s.started
}

func (s *stats) getLookups() uint64 {
	s.m.Lock()
	defer s.m.Unlock()

	return s.lookups
}

func (c *client) GetLookup(ctx context.Context, _ string) api.Lookup {
	c.bootup <- c.stats.bumpStarted()
	defer c.stats.bumpLookups()

	lookup := api.Lookup{}
	if c.failure == nil {
		lookup.Name = <-c.domain
	} else {
		lookup.Error = c.failure
	}

	return lookup
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

	entry := cache.store.LoadOrCreate(domain)

	if config.retrieved {
		entry.setResponse(api.Lookup{Name: domain})
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

			require.NoError(t, lookup.Error)
			require.Equal(t, "my.gitlab.com", lookup.Name)
			require.Equal(t, uint64(1), resolver.stats.getLookups())
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

			require.Equal(t, uint64(0), resolver.stats.getLookups())

			resolver.domain <- "my.gitlab.com"
			wg.Wait()

			require.Equal(t, uint64(1), resolver.stats.getLookups())
		})
	})

	t.Run("when item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: true}, func(*Entry) {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				require.Equal(t, "my.gitlab.com", lookup.Name)
				require.Equal(t, uint64(0), resolver.stats.getLookups())
			})
		})
	})

	t.Run("when a non-retrieved new item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(*Entry) {
				lookup := make(chan *api.Lookup, 1)

				go func() {
					lookup <- cache.Resolve(context.Background(), "my.gitlab.com")
				}()

				<-resolver.bootup

				require.Equal(t, uint64(1), resolver.stats.getStarted())
				require.Equal(t, uint64(0), resolver.stats.getLookups())

				resolver.domain <- "my.gitlab.com"
				<-lookup

				require.Equal(t, uint64(1), resolver.stats.getStarted())
				require.Equal(t, uint64(1), resolver.stats.getLookups())
			})
		})
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				require.Equal(t, "my.gitlab.com", lookup.Name)
				require.Equal(t, uint64(0), resolver.stats.getLookups())

				resolver.domain <- "my.gitlab.com"
				require.Equal(t, uint64(1), resolver.stats.getLookups())
			})
		})
	})

	t.Run("when item in long cache is requested multiple times", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")

				require.Equal(t, uint64(0), resolver.stats.getLookups())

				resolver.domain <- "my.gitlab.com"
				require.Equal(t, uint64(1), resolver.stats.getLookups())
			})
		})
	})

	t.Run("when retrieval failed with an error", func(t *testing.T) {
		withTestCache(resolverConfig{failure: errors.New("500 err")}, func(cache *Cache, resolver *client) {
			maxRetrievalInterval = 0

			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			require.Equal(t, uint64(3), resolver.stats.getLookups())
			require.EqualError(t, lookup.Error, "500 err")
		})
	})

	t.Run("when retrieval failed because of an external context being canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			lookup := cache.Resolve(ctx, "my.gitlab.com")

			require.Equal(t, uint64(0), resolver.stats.getLookups())
			require.EqualError(t, lookup.Error, "context done")
		})
	})

	t.Run("when retrieval failed because of an internal retriever context timeout", func(t *testing.T) {
		t.Skip("Data race")

		retrievalTimeout = 0
		defer func() { retrievalTimeout = 5 * time.Second }()

		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			require.Equal(t, uint64(0), resolver.stats.getLookups())
			require.EqualError(t, lookup.Error, "retrieval context done")
		})
	})

	t.Run("when retrieval failed because of resolution context being canceled", func(t *testing.T) {
		withTestCache(resolverConfig{}, func(cache *Cache, resolver *client) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(entry *Entry) {
				ctx, cancel := context.WithCancel(context.Background())

				response := make(chan *api.Lookup, 1)
				go func() { response <- cache.Resolve(ctx, "my.gitlab.com") }()

				cancel()

				resolver.domain <- "my.gitlab.com"
				lookup := <-response

				require.EqualError(t, lookup.Error, "context done")
			})
		})
	})
}
