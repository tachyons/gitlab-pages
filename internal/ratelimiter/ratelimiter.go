package ratelimiter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"gitlab.com/gitlab-org/gitlab-pages/internal/lru"
)

const (
	// DefaultSourceIPLimitPerSecond is the limit per second that rate.Limiter
	// needs to generate tokens every second.
	// The default value is 20 requests per second.
	DefaultSourceIPLimitPerSecond = 20.0
	// DefaultSourceIPBurstSize is the maximum burst allowed per rate limiter.
	// E.g. The first 100 requests within 1s will succeed, but the 101st will fail.
	DefaultSourceIPBurstSize = 100
)

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

// RateLimiter holds an LRU cache of elements to be rate limited. Currently, it supports
// a sourceIPCache and each item returns a rate.Limiter.
// It uses "golang.org/x/time/rate" as its Token Bucket rate limiter per source IP entry.
// See example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html
// It also holds a now function that can be mocked in unit tests.
type RateLimiter struct {
	now                    func() time.Time
	sourceIPLimitPerSecond float64
	sourceIPBurstSize      int
	sourceIPBlockedCount   *prometheus.GaugeVec
	sourceIPCache          *lru.Cache
	// TODO: add domainCache https://gitlab.com/gitlab-org/gitlab-pages/-/issues/630
}

// New creates a new RateLimiter with default values that can be configured via Option functions
func New(c *lru.Cache, opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		now:                    time.Now,
		sourceIPLimitPerSecond: DefaultSourceIPLimitPerSecond,
		sourceIPBurstSize:      DefaultSourceIPBurstSize,
		sourceIPCache:          c,
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}

// WithNow replaces the RateLimiter now function
func WithNow(now func() time.Time) Option {
	return func(rl *RateLimiter) {
		rl.now = now
	}
}

// WithSourceIPLimitPerSecond allows configuring per source IP limit per second for RateLimiter
func WithSourceIPLimitPerSecond(limit float64) Option {
	return func(rl *RateLimiter) {
		rl.sourceIPLimitPerSecond = limit
	}
}

// WithSourceIPBurstSize configures burst per source IP for the RateLimiter
func WithSourceIPBurstSize(burst int) Option {
	return func(rl *RateLimiter) {
		rl.sourceIPBurstSize = burst
	}
}

func WithBlockedCountMetric(m *prometheus.GaugeVec) Option {
	return func(rl *RateLimiter) {
		rl.sourceIPBlockedCount = m
	}
}

func (rl *RateLimiter) getSourceIPLimiter(sourceIP string) *rate.Limiter {
	limiterI, _ := rl.sourceIPCache.FindOrFetch(sourceIP, sourceIP, func() (interface{}, error) {
		return rate.NewLimiter(rate.Limit(rl.sourceIPLimitPerSecond), rl.sourceIPBurstSize), nil
	})

	return limiterI.(*rate.Limiter)
}

// SourceIPAllowed checks that the real remote IP address is allowed to perform an operation
func (rl *RateLimiter) SourceIPAllowed(sourceIP string) bool {
	limiter := rl.getSourceIPLimiter(sourceIP)

	// AllowN allows us to use the rl.now function, so we can test this more easily.
	return limiter.AllowN(rl.now(), 1)
}
