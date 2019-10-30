package gitlab

import (
	"time"

	cache "github.com/patrickmn/go-cache"
)

type Cache struct {
	shortCache *cache.Cache
	longCache  *cache.Cache
}

func NewCache() *Cache {
	return &Cache{
		shortCache: cache.New(5*time.Second, 10*time.Second),
		longCache:  cache.New(5*time.Minute, 10*time.Minute),
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
//  TODO use sync.Once to synchronize retrieval
func (c *Cache) GetLookup(domain string, retrieve func() *Lookup) *Lookup {
	return nil
}
