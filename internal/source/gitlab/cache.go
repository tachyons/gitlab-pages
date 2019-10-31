package gitlab

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// Cache is a short and long caching mechanism for GitLab source
type Cache struct {
	shortCache *cache.Cache
	longCache  *cache.Cache
}

type longEntry struct {
	lookup  *Lookup
	refresh *sync.Once
}

type retrieval struct {
	domain       string
	retries      int
	retrieveFunc func() (*Lookup, error)
}

var (
	maxRetrievalRetries  = 3
	maxRetrievalInterval = time.Minute
)

// NewCache creates a new instance of Cache and sets default expiration
func NewCache() *Cache {
	return &Cache{
		shortCache: cache.New(5*time.Second, time.Minute),
		longCache:  cache.New(10*time.Minute, time.Minute),
	}
}

// GetLookup is going to return a Lookup identified by a domain name using
// following algorithm:
// - if a domain lookup is present in the short cache it will return just it
// - if it is not present in a short cache it will check the long cache
// - if it is present in a long cache it will return the long cache version and
//   run an update in a separate thread that will fetch the lookup from the
//   GitLab source and replace the short and long cache entries
// - if a domain lookup is not present in the long cache we will fetch the
//   lookup from the domain source and client will need to wait
func (c *Cache) GetLookup(domain string, retriever func() (*Lookup, error)) *Lookup {
	// return lookup if it exists in the short cache
	if lookup, exists := c.shortCache.Get(domain); exists {
		return lookup.(*Lookup)
	}

	// return lookup it if exists in the long cache, schedule retrieval
	if long, exists := c.longCache.Get(domain); exists {
		entry := long.(*longEntry)

		entry.refresh.Do(func() {
			go c.retrieve(retrieval{
				domain:       domain,
				retrieveFunc: retriever,
			})
		})

		return entry.lookup
	}

	// make all clients wait and retrieve lookup synchronously
	// TODO once, wait for all clients
	return c.retrieve(retrieval{
		domain:       domain,
		retrieveFunc: retriever,
	})
}

func (c *Cache) store(domain string, lookup *Lookup) *Lookup {
	longCacheEntry := &longEntry{lookup: lookup, refresh: &sync.Once{}}

	c.shortCache.SetDefault(domain, lookup)
	c.longCache.SetDefault(domain, longCacheEntry)

	return lookup
}

func (c *Cache) retrieve(fetcher retrieval) *Lookup {
	lookup, err := fetcher.retrieveFunc()
	if err != nil {
		if fetcher.retries >= maxRetrievalRetries {
			return nil
		}

		time.Sleep(maxRetrievalInterval)

		return c.retrieve(retrieval{
			domain:       fetcher.domain,
			retries:      fetcher.retries + 1,
			retrieveFunc: fetcher.retrieveFunc,
		})
	}

	return c.store(fetcher.domain, lookup)
}
