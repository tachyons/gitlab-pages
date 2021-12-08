package ratelimiter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"gitlab.com/gitlab-org/gitlab-pages/internal/lru"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

const (
	// DefaultSourceIPLimitPerSecond is the limit per second that rate.Limiter
	// needs to generate tokens every second.
	// The default value is 20 requests per second.
	DefaultSourceIPLimitPerSecond = 20.0
	// DefaultSourceIPBurstSize is the maximum burst allowed per rate limiter.
	// E.g. The first 100 requests within 1s will succeed, but the 101st will fail.
	DefaultSourceIPBurstSize = 100

	// based on an avg ~4,000 unique IPs per minute
	// https://log.gprd.gitlab.net/app/lens#/edit/f7110d00-2013-11ec-8c8e-ed83b5469915?_g=h@e78830b
	DefaultSourceIPCacheSize = 5000
)

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

// KeyFunc returns unique identifier for the subject of rate limit(e.g. client IP or domain)
type KeyFunc func(*http.Request) string

// RateLimiter holds an LRU cache of elements to be rate limited.
// It uses "golang.org/x/time/rate" as its Token Bucket rate limiter per source IP entry.
// See example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html
// It also holds a now function that can be mocked in unit tests.
type RateLimiter struct {
	name           string
	now            func() time.Time
	limitPerSecond float64
	burstSize      int
	blockedCount   *prometheus.GaugeVec
	cache          *lru.Cache
	key            KeyFunc

	cacheOptions []lru.Option
}

// New creates a new RateLimiter with default values that can be configured via Option functions
func New(name string, opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		name:           name,
		now:            time.Now,
		limitPerSecond: DefaultSourceIPLimitPerSecond,
		burstSize:      DefaultSourceIPBurstSize,
		key:            request.GetRemoteAddrWithoutPort,
	}

	for _, opt := range opts {
		opt(rl)
	}

	rl.cache = lru.New(name, rl.cacheOptions...)

	return rl
}

// WithNow replaces the RateLimiter now function
func WithNow(now func() time.Time) Option {
	return func(rl *RateLimiter) {
		rl.now = now
	}
}

// WithLimitPerSecond allows configuring limit per second for RateLimiter
func WithLimitPerSecond(limit float64) Option {
	return func(rl *RateLimiter) {
		rl.limitPerSecond = limit
	}
}

// WithBurstSize configures burst per key for the RateLimiter
func WithBurstSize(burst int) Option {
	return func(rl *RateLimiter) {
		rl.burstSize = burst
	}
}

// WithBlockedCountMetric configures metric reporting how many requests were blocked
func WithBlockedCountMetric(m *prometheus.GaugeVec) Option {
	return func(rl *RateLimiter) {
		rl.blockedCount = m
	}
}

// WithCacheMaxSize configures cache size for ratelimiter
func WithCacheMaxSize(size int64) Option {
	return func(rl *RateLimiter) {
		rl.cacheOptions = append(rl.cacheOptions, lru.WithMaxSize(size))
	}
}

// WithCachedEntriesMetric configures metric reporting how many keys are currently stored in
// the rate-limiter cache
func WithCachedEntriesMetric(m *prometheus.GaugeVec) Option {
	return func(rl *RateLimiter) {
		rl.cacheOptions = append(rl.cacheOptions, lru.WithCachedEntriesMetric(m))
	}
}

// WithCachedRequestsMetric configures metric for how many times we ask key cache
func WithCachedRequestsMetric(m *prometheus.CounterVec) Option {
	return func(rl *RateLimiter) {
		rl.cacheOptions = append(rl.cacheOptions, lru.WithCachedRequestsMetric(m))
	}
}

func (rl *RateLimiter) limiter(key string) *rate.Limiter {
	limiterI, _ := rl.cache.FindOrFetch(key, key, func() (interface{}, error) {
		return rate.NewLimiter(rate.Limit(rl.limitPerSecond), rl.burstSize), nil
	})

	return limiterI.(*rate.Limiter)
}

// RequestAllowed checks that the real remote IP address is allowed to perform an operation
func (rl *RateLimiter) RequestAllowed(r *http.Request) bool {
	rateLimitedKey := rl.key(r)
	limiter := rl.limiter(rateLimitedKey)

	// AllowN allows us to use the rl.now function, so we can test this more easily.
	return limiter.AllowN(rl.now(), 1)
}
