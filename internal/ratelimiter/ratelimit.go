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

	//example https://www.fatalerrors.org/a/design-and-implementation-of-time-rate-limiter-for-golang-standard-library.html

	// DefaultRatePerDomainPerSecond the maximum number of requests per second to be allowed per domain
	// 1 request every 25ms = 40 rps
	DefaultRatePerDomainPerSecond = 25 * time.Millisecond
	// DefaultPerDomainMaxBurstPerSecond is the maximum burst allowed per rate limiter
	// 40 items in the bucket is the max
	// so if there are 40 rquests in 25 milliseconds they will succeed, but request 41st will fail
	DefaultPerDomainMaxBurstPerSecond = 40
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
	domainRatePerSecond     time.Duration
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

func WithDomainRatePerSecond(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.domainRatePerSecond = d
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
			// the first argument is the number of requests per second that will be allowed,
			limiter: rate.NewLimiter(rate.Every(rl.domainRatePerSecond), rl.perDomainBurstPerSecond),
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
	defer func() {
		fmt.Printf("was domain: %q allowed? - %t\n", domain, res)
	}()

	counter := rl.getDomainCounter(domain)
	counter.limiter.Reserve()
	fmt.Printf("COUNTER DETAILS? now: %s :limit: %f burst: %d\n", rl.now(), counter.limiter.Limit(), counter.limiter.Burst())
	counter.limiter.Burst()
	// TODO: we could use Wait(ctx) if we want to moderate the request rate rather than denying requests
	return counter.limiter.AllowN(rl.now(), 1)
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
