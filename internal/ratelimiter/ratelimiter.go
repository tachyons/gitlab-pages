package ratelimiter

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"gitlab.com/gitlab-org/gitlab-pages/internal/lru"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

const (
	// based on an avg ~4,000 unique IPs per minute
	// https://log.gprd.gitlab.net/app/lens#/edit/f7110d00-2013-11ec-8c8e-ed83b5469915?_g=h@e78830b
	DefaultSourceIPCacheSize = 5000

	// we have less than 4000 different hosts per minute
	// https://log.gprd.gitlab.net/app/dashboards#/view/d52ab740-61a4-11ec-b20d-65f14d890d9b?_a=(viewMode:edit)&_g=h@42b0d52
	DefaultDomainCacheSize = 4000
)

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

// KeyFunc returns unique identifier for the subject of rate limit(e.g. client IP or domain)
type KeyFunc func(*http.Request) string

// TLSKeyFunc is used by GetCertificateMiddleware to identify the subject of rate limit (client IP or SNI servername)
type TLSKeyFunc func(*tls.ClientHelloInfo) string

// RateLimiter holds an LRU cache of elements to be rate limited.
// It uses "golang.org/x/time/rate" as its Token Bucket rate limiter per source IP entry.
// See example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html
// It also holds a now function that can be mocked in unit tests.
type RateLimiter struct {
	name           string
	now            func() time.Time
	keyFunc        KeyFunc
	tlsKeyFunc     TLSKeyFunc
	limitPerSecond float64
	burstSize      int
	blockedCount   *prometheus.GaugeVec
	cache          *lru.Cache

	cacheOptions []lru.Option
}

// New creates a new RateLimiter with default values that can be configured via Option functions
func New(name string, opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		name:    name,
		now:     time.Now,
		keyFunc: request.GetRemoteAddrWithoutPort,
	}

	for _, opt := range opts {
		opt(rl)
	}

	if rl.limitPerSecond > 0.0 {
		rl.cache = lru.New(name, rl.cacheOptions...)
	}

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

// WithBurstSize configures burst per keyFunc value for the RateLimiter
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

// WithCachedRequestsMetric configures metric for how many times we access cache
func WithCachedRequestsMetric(m *prometheus.CounterVec) Option {
	return func(rl *RateLimiter) {
		rl.cacheOptions = append(rl.cacheOptions, lru.WithCachedRequestsMetric(m))
	}
}

// WithKeyFunc configures keyFunc
func WithKeyFunc(f KeyFunc) Option {
	return func(rl *RateLimiter) {
		rl.keyFunc = f
	}
}

func TLSHostnameKey(info *tls.ClientHelloInfo) string {
	return info.ServerName
}

func TLSClientIPKey(info *tls.ClientHelloInfo) string {
	remoteAddr := info.Conn.RemoteAddr().String()
	remoteAddr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return remoteAddr
}

func WithTLSKeyFunc(keyFunc TLSKeyFunc) Option {
	return func(rl *RateLimiter) {
		rl.tlsKeyFunc = keyFunc
	}
}

func (rl *RateLimiter) limiter(key string) *rate.Limiter {
	limiterI, _ := rl.cache.FindOrFetch(key, key, func() (interface{}, error) {
		return rate.NewLimiter(rate.Limit(rl.limitPerSecond), rl.burstSize), nil
	})

	return limiterI.(*rate.Limiter)
}

// requestAllowed checks if request is within the rate-limit
func (rl *RateLimiter) requestAllowed(r *http.Request) bool {
	rateLimitedKey := rl.keyFunc(r)

	return rl.allowed(rateLimitedKey)
}

func (rl *RateLimiter) allowed(rateLimitedKey string) bool {
	limiter := rl.limiter(rateLimitedKey)

	// AllowN allows us to use the rl.now function, so we can test this more easily.
	return limiter.AllowN(rl.now(), 1)
}
