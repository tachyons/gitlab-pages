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

// based on an avg ~4,000 unique IPs per minute
// https://log.gprd.gitlab.net/app/lens#/edit/f7110d00-2013-11ec-8c8e-ed83b5469915?_g=h@e78830b
const defaultCacheMaxSize = 1000
const defaultCacheExpirationInterval = time.Minute

// Option function to configure a Cache
type Option func(*Cache)

// Cache wraps a ccache and allows setting custom metrics for hits/misses.
type Cache struct {
	op                  string
	duration            time.Duration
	maxSize             int64
	cache               *ccache.Cache
	metricCachedEntries *prometheus.GaugeVec
	metricCacheRequests *prometheus.CounterVec
}

// New creates an LRU cache
func New(op string, opts ...Option) *Cache {
	c := &Cache{
		op:       op,
		duration: defaultSourceIPExpirationInterval,
		maxSize:  defaultSourceIPItems,
	}

	for _, opt := range opts {
		opt(c)
	}

	configuration := ccache.Configure()
	configuration.MaxSize(c.maxSize)
	configuration.ItemsToPrune(uint32(c.maxSize) / itemsToPruneDiv)
	configuration.GetsPerPromote(getsPerPromote) // if item gets requested frequently promote it
	configuration.OnDelete(func(*ccache.Item) {
		if c.metricCachedEntries != nil {
			c.metricCachedEntries.WithLabelValues(op).Dec()
		}
	})

	c.cache = ccache.New(configuration)

	return c
}

// FindOrFetch will try to get the item from the cache if exists and is not expired.
// If it can't find it, it will call fetchFn to retrieve the item and cache it.
func (c *Cache) FindOrFetch(cacheNamespace, key string, fetchFn func() (interface{}, error)) (interface{}, error) {
	item := c.cache.Get(cacheNamespace + key)

	if item != nil && !item.Expired() {
		if c.metricCacheRequests != nil {
			c.metricCacheRequests.WithLabelValues(c.op, "hit").Inc()
		}
		return item.Value(), nil
	}

	value, err := fetchFn()
	if err != nil {
		if c.metricCacheRequests != nil {
			c.metricCacheRequests.WithLabelValues(c.op, "error").Inc()
		}
		return nil, err
	}

	if c.metricCacheRequests != nil {
		c.metricCacheRequests.WithLabelValues(c.op, "miss").Inc()
	}
	if c.metricCachedEntries != nil {
		c.metricCachedEntries.WithLabelValues(c.op).Inc()
	}

	c.cache.Set(cacheNamespace+key, value, c.duration)

	return value, nil
}

func WithCachedEntriesMetric(m *prometheus.GaugeVec) Option {
	return func(c *Cache) {
		c.metricCachedEntries = m
	}
}

func WithCachedRequestsMetric(m *prometheus.CounterVec) Option {
	return func(c *Cache) {
		c.metricCacheRequests = m
	}
}

func WithExpirationInterval(t time.Duration) Option {
	return func(c *Cache) {
		c.duration = t
	}
}

func WithMaxSize(i int64) Option {
	return func(c *Cache) {
		c.maxSize = i
	}
}
