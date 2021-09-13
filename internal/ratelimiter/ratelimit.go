package ratelimiter

import (
	"fmt"
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

	// DefaultRatePerDomainPerSecond transformed to rate.Limit = 1 / DefaultRatePerDomainPerSecond.
	// The default value is equivalent to 100 requests per second per domain
	DefaultRatePerDomainPerSecond = 0.01
	// DefaultPerDomainMaxBurstPerSecond is the maximum burst in requests. TODO need to understand and test this
	DefaultPerDomainMaxBurstPerSecond = 100
)

type counter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Option function to configure a RateLimiter
type Option func(*RateLimiter)

type RateLimiter struct {
	now                     func() time.Time
	cleanupTimer            *time.Ticker
	maxTimePerDomain        time.Duration
	domainRatePerSecond     float64
	perDomainBurstPerSecond int
	domainMux               *sync.RWMutex
	// TODO: this could be an LRU cache like what we do in the zip VFS instead of cleaning manually ?
	perDomain map[string]*counter
}

// New creates a new RateLimiter with default values
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		now:                     time.Now,
		cleanupTimer:            time.NewTicker(DefaultCleanupInterval),
		maxTimePerDomain:        DefaultMaxTimePerDomain,
		domainRatePerSecond:     DefaultRatePerDomainPerSecond,
		perDomainBurstPerSecond: DefaultPerDomainMaxBurstPerSecond,
		domainMux:               &sync.RWMutex{},
		perDomain:               make(map[string]*counter),
	}

	for _, opt := range opts {
		opt(rl)
	}

	go rl.cleanup()

	return rl
}

func WithNow(now func() time.Time) Option {
	return func(rl *RateLimiter) {
		rl.now = now
	}
}

func WithCleanupInterval(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.cleanupTimer.Reset(d)
	}
}

func WithDomainRatePerSecond(r float64) Option {
	return func(rl *RateLimiter) {
		rl.domainRatePerSecond = r
	}
}

func WithDomainBurstPerSecond(burst int) Option {
	return func(rl *RateLimiter) {
		rl.perDomainBurstPerSecond = burst
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
			limiter:  rate.NewLimiter(rate.Limit(rl.domainRatePerSecond), rl.perDomainBurstPerSecond),
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
	defer func() {
		fmt.Printf("limiter info: limit: %f - burst: %d\n", counter.limiter.Limit(), counter.limiter.Burst())
		fmt.Printf("calling DomainAllowed for: %q returned: %t\n", domain, res)
	}()

	// TODO: we could use Wait(ctx) if we want to moderate the request rate rather than denying requests
	return counter.limiter.Allow()
}

func (rl *RateLimiter) cleanup() {
	select {
	case t := <-rl.cleanupTimer.C:
		log.WithField("cleanup", t).Debug("cleaning perDomain rate")

		rl.domainMux.Lock()
		for domain, counter := range rl.perDomain {
			if time.Since(counter.lastSeen) > rl.maxTimePerDomain {
				delete(rl.perDomain, domain)
			}
		}
		rl.domainMux.Unlock()
	default:
	}
}
