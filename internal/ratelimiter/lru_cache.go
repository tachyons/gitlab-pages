package ratelimiter

import (
	"time"

	"github.com/karlseguin/ccache/v2"
)

// lruCacheGetPerPromote is a value that makes the item to be promoted
// it is taken arbitrarily as a sane value indicating that the item
// was frequently picked
// promotion moves the item to the front of the LRU list
const lruCacheGetsPerPromote = 64

// lruCacheItemsToPruneDiv is a value that indicates how many items
// need to be pruned on OOM, this prunes 1/16 of items
const lruCacheItemsToPruneDiv = 16

type lruCache struct {
	op       string
	duration time.Duration
	cache    *ccache.Cache
}

func newLruCache(op string, maxEntries int64, duration time.Duration) *lruCache {
	configuration := ccache.Configure()
	configuration.MaxSize(maxEntries)
	configuration.ItemsToPrune(uint32(maxEntries) / lruCacheItemsToPruneDiv)
	configuration.GetsPerPromote(lruCacheGetsPerPromote) // if item gets requested frequently promote it
	configuration.OnDelete(func(*ccache.Item) {
		// TODO: add metrics
		//metrics.ZipCachedEntries.WithLabelValues(op).Dec()
	})

	return &lruCache{
		op:       op,
		cache:    ccache.New(configuration),
		duration: duration,
	}
}

func (c *lruCache) findOrFetch(cacheNamespace, key string, fetchFn func() (interface{}, error)) (interface{}, error) {
	item := c.cache.Get(cacheNamespace + key)

	if item != nil && !item.Expired() {
		// TODO: add metrics
		//metrics.ZipCacheRequests.WithLabelValues(c.op, "hit").Inc()
		return item.Value(), nil
	}

	value, err := fetchFn()
	if err != nil {
		// TODO: add metrics
		//metrics.ZipCacheRequests.WithLabelValues(c.op, "error").Inc()
		return nil, err
	}

	// TODO: add metrics
	//metrics.ZipCacheRequests.WithLabelValues(c.op, "miss").Inc()
	//metrics.ZipCachedEntries.WithLabelValues(c.op).Inc()

	c.cache.Set(cacheNamespace+key, value, c.duration)
	return value, nil
}
