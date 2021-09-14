package ratelimiter

import (
	"sync"
	"time"

	"gitlab.com/gitlab-org/labkit/log"
	"golang.org/x/time/rate"
)

const (
	// DefaultCleanupInterval is the time at which cleanup will run
	DefaultCleanupInterval = 30 * time.Second
	// DefaultMaxTimePerDomain is the maximum time to keep a domain in the rate limiter map
	DefaultMaxTimePerDomain = 30 * time.Second

	// DefaultPerDomainFrequency the maximum number of requests per second to be allowed per domain.
	// The default value of 25ms equals 1 request every 25ms -> 40 rps
	DefaultPerDomainFrequency = 25 * time.Millisecond
	// DefaultPerDomainBurstSize is the maximum burst allowed per rate limiter
	// E.g. The first 40 requests within 25ms will succeed, but the 41st will fail until the next
	// refill occurs at DefaultPerDomainFrequency, allowing only 1 request per rate frequency.
	DefaultPerDomainBurstSize = 40
)

type counter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

// RateLimiter holds a map ot domain names with counters that enable rate limiting per domain.
// It uses "golang.org/x/time/rate" as its Token Bucket rate limiter per domain entry.
// See example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html
// Cleanup runs every cleanupTimer iteration over all domains and removing them if
// the time since counter.lastSeen is greater than the domainMaxTTL.
type RateLimiter struct {
	now                func() time.Time
	cleanupTimer       *time.Ticker
	domainMaxTTL       time.Duration
	perDomainFrequency time.Duration
	perDomainBurstSize int
	domainMux          *sync.RWMutex
	// TODO: this could be an LRU cache like what we do in the zip VFS instead of cleaning manually ?
	perDomain map[string]*counter
}

// New creates a new RateLimiter with default values that can be configured via Option functions
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		now:                time.Now,
		cleanupTimer:       time.NewTicker(DefaultCleanupInterval),
		domainMaxTTL:       DefaultMaxTimePerDomain,
		perDomainFrequency: DefaultPerDomainFrequency,
		perDomainBurstSize: DefaultPerDomainBurstSize,
		domainMux:          &sync.RWMutex{},
		perDomain:          make(map[string]*counter),
	}

	for _, opt := range opts {
		opt(rl)
	}

	go rl.cleanup()

	return rl
}

// WithNow replaces the RateLimiter now function
func WithNow(now func() time.Time) Option {
	return func(rl *RateLimiter) {
		rl.now = now
	}
}

// WithCleanupInterval replaces the RateLimiter cleanup interval
func WithCleanupInterval(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.cleanupTimer.Reset(d)
	}
}

// WithPerDomainFrequency allows configuring perDomain frequency for the RateLimiter
func WithPerDomainFrequency(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.perDomainFrequency = d
	}
}

// WithPerDomainBurstSize configures burst per domain for the RateLimiter
func WithPerDomainBurstSize(burst int) Option {
	return func(rl *RateLimiter) {
		rl.perDomainBurstSize = burst
	}
}

func (rl *RateLimiter) getDomainCounter(domain string) *counter {
	rl.domainMux.Lock()
	defer rl.domainMux.Unlock()

	// TODO: add metrics
	currentCounter, ok := rl.perDomain[domain]
	if !ok {
		newCounter := &counter{
			lastSeen: rl.now(),
			// the first argument is the number of requests per second that will be allowed,
			limiter: rate.NewLimiter(rate.Every(rl.perDomainFrequency), rl.perDomainBurstSize),
		}

		rl.perDomain[domain] = newCounter
		return newCounter
	}

	currentCounter.lastSeen = rl.now()
	return currentCounter
}

// DomainAllowed checks that the requested domain can be accessed within
// the maxCountPerDomain in the given domainWindow.
func (rl *RateLimiter) DomainAllowed(domain string) (res bool) {
	counter := rl.getDomainCounter(domain)

	// TODO: we could use Wait(ctx) if we want to moderate the request rate rather than denying requests
	return counter.limiter.AllowN(rl.now(), 1)
}

func (rl *RateLimiter) cleanup() {
	select {
	case t := <-rl.cleanupTimer.C:
		log.WithField("cleanup", t).Debug("cleaning perDomain rate")

		rl.domainMux.Lock()
		for domain, counter := range rl.perDomain {
			if time.Since(counter.lastSeen) > rl.domainMaxTTL {
				delete(rl.perDomain, domain)
			}
		}
		rl.domainMux.Unlock()
	default:
	}
}
