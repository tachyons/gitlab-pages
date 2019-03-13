package client

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

type cachedDomainResponse struct {
	host     string
	response *DomainResponse
	err      error

	once sync.Once
}

type cachedAPI struct {
	upstream        API
	positiveTimeout time.Duration
	negativeTimeout time.Duration

	cache *cache.Cache
}

func (a *cachedAPI) ensureRequestDomain(c *cachedDomainResponse) {
	c.once.Do(func() {
		c.response, c.err = a.upstream.RequestDomain(c.host)

		// cache entry
		if c.err == nil {
			a.cache.Set(c.host, c, a.positiveTimeout)
		} else {
			a.cache.Set(c.host, c, a.negativeTimeout)
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

func (a *cachedAPI) newCacheEntry(host string) *cachedDomainResponse {
	cachedObject := &cachedDomainResponse{host: host}
	a.cache.Set(cachedObject.host, cachedObject, a.negativeTimeout)
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
	return cachedObject.response, cachedObject.err
}

func (a *cachedAPI) IsReady() bool {
	return a.upstream.IsReady()
}

func NewCachedClient(upstream API, positiveTimeout time.Duration, negativeTimeout time.Duration) API {
	return &cachedAPI{
		upstream:        upstream,
		positiveTimeout: positiveTimeout,
		negativeTimeout: negativeTimeout,
		cache:           cache.New(positiveTimeout, negativeTimeout),
	}
}
