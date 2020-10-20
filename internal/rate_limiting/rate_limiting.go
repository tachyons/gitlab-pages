package rate_limiting

import (
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type rateLimit struct {
	*rate.Limiter
}

type RateLimiting struct {
	cache *cache.Cache

	window time.Duration
	limit  uint
}

func NewRateLimiting(window time.Duration, limit uint) *RateLimiting {
	return &RateLimiting{
		cache:  cache.New(window*2, window),
		window: window,
		limit:  limit,
	}
}

func (r *RateLimiting) newRateLimiter() rateLimit {
	// we divide a window by amount of requests
	// the bucket is refilled every interval
	// allowing to consume up to the defined `limit`
	everyNs := r.window.Nanoseconds() / int64(r.limit)
	every := time.Duration(everyNs)

	return rateLimit{
		rate.NewLimiter(rate.Every(every), int(r.limit)),
	}
}

func (r *RateLimiting) findOrCreate(key string) rateLimit {
	for {
		// try to get existing item
		if item, expiry, found := r.cache.GetWithExpiration(key); found {
			// extend item window
			if time.Until(expiry) > r.window {
				r.cache.SetDefault(key, item)
			}

			return item.(rateLimit)
		}

		// add a new item
		if rateLimiter := r.newRateLimiter(); r.cache.Add(key, rateLimiter, cache.DefaultExpiration) == nil {
			return rateLimiter
		}
	}
}

func (r *RateLimiting) Allow(key string) bool {
	return r.findOrCreate(key).Allow()
}
