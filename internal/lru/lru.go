package lru

import (
	"time"

	"github.com/karlseguin/ccache/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// lruCacheGetPerPromote is a value that makes the item to be promoted
// it is taken arbitrally as a sane value indicating that the item
// was frequently picked
// promotion moves the item to the front of the LRU list
const getsPerPromote = 64

// itemsToPruneDiv is a value that indicates how much items
// needs to be pruned on OOM, this prunes 1/16 of items
const itemsToPruneDiv = 16

// Cache wraps a ccache and allows setting custom metrics for hits/misses.
type Cache struct {
	op                  string
	duration            time.Duration
	cache               *ccache.Cache
	metricCachedEntries *prometheus.GaugeVec
	metricCacheRequests *prometheus.CounterVec
}

// New creates an LRU cache
func New(op string, maxEntries int64, duration time.Duration, cachedEntriesMetric *prometheus.GaugeVec, cacheRequestsMetric *prometheus.CounterVec) *Cache {
	configuration := ccache.Configure()
	configuration.MaxSize(maxEntries)
	configuration.ItemsToPrune(uint32(maxEntries) / itemsToPruneDiv)
	configuration.GetsPerPromote(getsPerPromote) // if item gets requested frequently promote it
	configuration.OnDelete(func(*ccache.Item) {
		cachedEntriesMetric.WithLabelValues(op).Dec()
	})

	return &Cache{
		op:                  op,
		cache:               ccache.New(configuration),
		duration:            duration,
		metricCachedEntries: cachedEntriesMetric,
		metricCacheRequests: cacheRequestsMetric,
	}
}

// FindOrFetch will try to get the item from the cache if exists and is not expired.
// If it can't find it, it will call fetchFn to retrieve the item and cache it.
func (c *Cache) FindOrFetch(cacheNamespace, key string, fetchFn func() (interface{}, error)) (interface{}, error) {
	item := c.cache.Get(cacheNamespace + key)

	if item != nil && !item.Expired() {
		c.metricCacheRequests.WithLabelValues(c.op, "hit").Inc()
		return item.Value(), nil
	}

	value, err := fetchFn()
	if err != nil {
		c.metricCacheRequests.WithLabelValues(c.op, "error").Inc()
		return nil, err
	}

	c.metricCacheRequests.WithLabelValues(c.op, "miss").Inc()
	c.metricCachedEntries.WithLabelValues(c.op).Inc()

	c.cache.Set(cacheNamespace+key, value, c.duration)

	return value, nil
}
