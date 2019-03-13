package client

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

const refreshCacheInterval = 3 * time.Second
const defaultCacheTimeout = 3 * time.Second

type cachedDomainResponse struct {
	host     string
	response *DomainResponse
	err      error

	once sync.Once
}

func (c *cachedDomainResponse) log() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"host": c.host,
	})
}

// cachedAPI implements a cache layer for all requests
// the request is executed exactly once for all clients
// we store positive results in `cache` for cacheTimeout interval
// we also store negative results in `cache` for time defined by defaultCacheTimeout
// to solve temporary API failures we retain last successful result
// for time specified in `longCacheTimeout` and use it as last resort
// this makes us to request domain config every `cacheTimeout` in case of found domains
// and request every `defaultCacheTimeout` if there's API failure
type cachedAPI struct {
	upstream         API
	cacheTimeout     time.Duration
	longCacheTimeout time.Duration

	cache     *cache.Cache
	longCache *cache.Cache
}

func (a *cachedAPI) ensureRequestDomain(c *cachedDomainResponse) {
	c.once.Do(func() {
		c.response, c.err = a.upstream.RequestDomain(c.host)
		c.log().WithError(c.err).Debugln("CachedRequestDomain")

		// add positive result to cache and in long cache for longer period
		if c.err == nil {
			a.cache.Set(c.host, c, a.cacheTimeout)
			a.longCache.Set(c.host, c, a.longCacheTimeout)
		} else {
			a.cache.Set(c.host, c, cache.DefaultExpiration)
		}
	})
}

func (a *cachedAPI) findCacheEntry(host string) *cachedDomainResponse {
	// try to get object from cache
	if cached, found := a.cache.Get(host); found {
		return cached.(*cachedDomainResponse)
	}

	return nil
}

func (a *cachedAPI) findLongCacheEntry(host string) *cachedDomainResponse {
	// try to get object from cache
	if cached, found := a.longCache.Get(host); found {
		return cached.(*cachedDomainResponse)
	}

	return nil
}

func (a *cachedAPI) newCacheEntry(host string) *cachedDomainResponse {
	cachedObject := &cachedDomainResponse{host: host}

	// cache object for short period
	a.cache.Set(cachedObject.host, cachedObject, cache.DefaultExpiration)
	return cachedObject
}

// RequestDomain request a host from preconfigured list of domains
func (a *cachedAPI) RequestDomain(host string) (*DomainResponse, error) {
	cachedObject := a.findCacheEntry(host)

	// create a new cache entry
	if cachedObject == nil {
		cachedObject = a.newCacheEntry(host)
	}

	// request or wait for API response
	a.ensureRequestDomain(cachedObject)

	// try to take from long cache to ignore short failures
	if cachedObject.err != nil {
		cachedObject = a.findLongCacheEntry(host)
	}

	return cachedObject.response, cachedObject.err
}

func (a *cachedAPI) IsReady() bool {
	return a.upstream.IsReady()
}

func NewCachedClient(upstream API, cacheTimeout, longCacheTimeout time.Duration) API {
	return &cachedAPI{
		upstream:         upstream,
		cacheTimeout:     cacheTimeout,
		longCacheTimeout: longCacheTimeout,
		cache:            cache.New(defaultCacheTimeout, refreshCacheInterval),
		longCache:        cache.New(defaultCacheTimeout, refreshCacheInterval),
	}
}
