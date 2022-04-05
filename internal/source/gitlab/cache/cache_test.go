package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

var testCacheConfig = config.Cache{
	CacheExpiry:          time.Second,
	CacheCleanupInterval: time.Second / 2,
	EntryRefreshTimeout:  time.Second / 2,
	RetrievalTimeout:     time.Second,
	MaxRetrievalInterval: time.Second / 3,
	MaxRetrievalRetries:  3,
}

type clientMock struct {
	counter uint64
	lookups chan uint64
	domain  chan string
	failure error
}

func (c *clientMock) GetLookup(ctx context.Context, _ string) api.Lookup {
	lookup := api.Lookup{}
	if c.failure == nil {
		lookup.Name = <-c.domain
	} else {
		lookup.Error = c.failure
	}

	c.lookups <- atomic.AddUint64(&c.counter, 1)

	return lookup
}

func (c *clientMock) Status() error {
	return nil
}

func withTestCache(config resolverConfig, cacheConfig *config.Cache, block func(*Cache, *clientMock)) {
	var chanSize int

	if config.buffered {
		chanSize = 1
	} else {
		chanSize = 0
	}

	resolver := &clientMock{
		domain:  make(chan string, chanSize),
		lookups: make(chan uint64, 100),
		failure: config.failure,
	}
	if cacheConfig == nil {
		cacheConfig = &testCacheConfig
	}

	cache := NewCache(resolver, cacheConfig)

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
		withTestCache(resolverConfig{buffered: true}, nil, func(cache *Cache, resolver *clientMock) {
			require.Empty(t, resolver.lookups)
			resolver.domain <- "my.gitlab.com"

			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			require.NoError(t, lookup.Error)
			require.Equal(t, "my.gitlab.com", lookup.Name)
			require.Equal(t, uint64(1), <-resolver.lookups)
		})
	})

	t.Run("when item is not cached and accessed multiple times", func(t *testing.T) {
		withTestCache(resolverConfig{}, nil, func(cache *Cache, resolver *clientMock) {
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

			require.Empty(t, resolver.lookups)

			resolver.domain <- "my.gitlab.com"
			wg.Wait()

			require.Equal(t, uint64(1), <-resolver.lookups)
		})
	})

	t.Run("when item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, nil, func(cache *Cache, resolver *clientMock) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: true}, func(*Entry) {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				require.Equal(t, "my.gitlab.com", lookup.Name)
				require.Empty(t, resolver.lookups)
			})
		})
	})

	t.Run("when a non-retrieved new item is in short cache", func(t *testing.T) {
		withTestCache(resolverConfig{}, nil, func(cache *Cache, resolver *clientMock) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(*Entry) {
				lookup := make(chan *api.Lookup, 1)

				go func() {
					lookup <- cache.Resolve(context.Background(), "my.gitlab.com")
				}()

				require.Empty(t, resolver.lookups)

				resolver.domain <- "my.gitlab.com"
				<-lookup

				require.Equal(t, uint64(1), <-resolver.lookups)
			})
		})
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		withTestCache(resolverConfig{buffered: false}, nil, func(cache *Cache, resolver *clientMock) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
				lookup := cache.Resolve(context.Background(), "my.gitlab.com")

				require.Equal(t, "my.gitlab.com", lookup.Name)
				require.Empty(t, resolver.lookups)

				resolver.domain <- "my.gitlab.com"

				require.Equal(t, uint64(1), <-resolver.lookups)
			})
		})
	})

	t.Run("when item in long cache is requested multiple times", func(t *testing.T) {
		withTestCache(resolverConfig{}, nil, func(cache *Cache, resolver *clientMock) {
			cache.withTestEntry(entryConfig{expired: true, retrieved: true}, func(*Entry) {
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")
				cache.Resolve(context.Background(), "my.gitlab.com")

				require.Empty(t, resolver.lookups)

				resolver.domain <- "my.gitlab.com"

				require.Equal(t, uint64(1), <-resolver.lookups)
			})
		})
	})

	t.Run("when retrieval failed with an error", func(t *testing.T) {
		cc := testCacheConfig
		cc.MaxRetrievalInterval = 0
		err := errors.New("500 error")

		withTestCache(resolverConfig{failure: err}, &cc, func(cache *Cache, resolver *clientMock) {
			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			require.Len(t, resolver.lookups, 3)
			require.EqualError(t, lookup.Error, "500 error")
		})
	})

	t.Run("when retrieval failed because of an internal retriever context timeout", func(t *testing.T) {
		cc := testCacheConfig
		cc.RetrievalTimeout = 0

		withTestCache(resolverConfig{}, &cc, func(cache *Cache, resolver *clientMock) {
			lookup := cache.Resolve(context.Background(), "my.gitlab.com")

			require.Empty(t, resolver.lookups)
			require.ErrorIs(t, lookup.Error, context.DeadlineExceeded)
		})
	})

	t.Run("when retrieval failed because of resolution context being canceled", func(t *testing.T) {
		withTestCache(resolverConfig{}, nil, func(cache *Cache, resolver *clientMock) {
			cache.withTestEntry(entryConfig{expired: false, retrieved: false}, func(entry *Entry) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				lookup := cache.Resolve(ctx, "my.gitlab.com")
				resolver.domain <- "err.gitlab.com"

				require.Equal(t, "my.gitlab.com", lookup.Name)
				require.ErrorIs(t, lookup.Error, context.Canceled)
			})
		})
	})
}
