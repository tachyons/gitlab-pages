package zip

import (
	"time"

	"github.com/karlseguin/ccache/v2"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

type lruCache struct {
	op       string
	duration time.Duration
	cache    *ccache.Cache
}

func newLruCache(op string, maxEntries uint32, duration time.Duration) *lruCache {
	configuration := ccache.Configure()
	configuration.MaxSize(int64(maxEntries))
	configuration.ItemsToPrune(maxEntries / 16)
	configuration.GetsPerPromote(64) // if item gets requested frequently promote it
	configuration.OnDelete(func(*ccache.Item) {
		metrics.ZipCachedEntries.WithLabelValues(op).Dec()
	})

	return &lruCache{
		cache:    ccache.New(configuration),
		duration: duration,
	}
}

func (c *lruCache) findOrFetch(namespace, key string, fetchFn func() (interface{}, error)) (interface{}, error) {
	item := c.cache.Get(namespace + key)

	if item != nil && !item.Expired() {
		metrics.ZipCacheRequests.WithLabelValues(c.op, "hit").Inc()
		return item.Value(), nil
	}

	value, err := fetchFn()
	if err != nil {
		metrics.ZipCacheRequests.WithLabelValues(c.op, "error").Inc()
		return nil, err
	}

	metrics.ZipCacheRequests.WithLabelValues(c.op, "miss").Inc()
	metrics.ZipCachedEntries.WithLabelValues(c.op).Inc()

	c.cache.Set(namespace+key, value, c.duration)
	return value, nil
}
