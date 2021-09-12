package ratelimiter

import (
	"errors"
	"sync"
	"time"

	"gitlab.com/gitlab-org/labkit/log"
)

const (
	DefaultCleanupInterval   = time.Second
	DefaultWindowPerDomain   = time.Second
	DefaultPerDomainMaxCount = 100
)

var (
	errDomainCounterNotFound = errors.New("domain counter not found")
)

type counter struct {
	count    int64
	lastSeen time.Time
}

type Option func(*RateLimiter)

type RateLimiter struct {
	now               func() time.Time
	cleanupTimer      *time.Ticker
	domainWindow      time.Duration
	maxCountPerDomain int64
	domainMux         *sync.RWMutex
	// TODO: this could be an LRU cache like what we do in the zip VFS
	perDomain map[string]counter
}

// New creates a new RateLimiter with default values
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		now:               time.Now,
		cleanupTimer:      time.NewTicker(DefaultCleanupInterval),
		domainWindow:      DefaultWindowPerDomain,
		maxCountPerDomain: DefaultPerDomainMaxCount,
		domainMux:         &sync.RWMutex{},
		perDomain:         make(map[string]counter),
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

func WithDomainWindow(d time.Duration) Option {
	return func(rl *RateLimiter) {
		rl.domainWindow = d
	}
}
func WithDomainMaxCount(c int64) Option {
	return func(rl *RateLimiter) {
		rl.maxCountPerDomain = c
	}
}

// AddDomain to the current RateLimiter per domain count
func (rl *RateLimiter) AddDomain(domain string) {
	rl.domainMux.Lock()
	defer rl.domainMux.Unlock()

	// TODO: add metrics
	currentCounter, ok := rl.perDomain[domain]
	if !ok {
		newCounter := counter{
			lastSeen: rl.now(),
			count:    1,
		}

		rl.perDomain[domain] = newCounter
		return
	}

	currentCounter.count++
}

// DomainAllowed checks that the requested domain can be accessed within
// the maxCountPerDomain in the given domainWindow.
func (rl *RateLimiter) DomainAllowed(domain string) bool {
	// increment counter for this domain regardless if allowed or not
	defer rl.AddDomain(domain)

	domainCounter, err := rl.getDomainCounter(domain)
	if err != nil && errors.Is(err, errDomainCounterNotFound) {
		// we haven't seen this domain so it should be allowed
		log.WithError(err).Warn("DomainAllowed did not find the requested domain")
		return true
	}

	now := rl.now()
	lastSeen := domainCounter.lastSeen
	count := domainCounter.count

	//if requested within time window and the count is less thant the max count
	// e.g. maxCount = 10 and window is 10s
	// now is 1s, count is 1 -> true
	// now is 11s, count is < 10 -> true
	// now is 2s, count > 10 -> false
	if now.Sub(lastSeen) < rl.domainWindow {
		if count < rl.maxCountPerDomain {
			return true
		}
	}

	return false
}

func (rl *RateLimiter) getDomainCounter(domain string) (counter, error) {
	rl.domainMux.RLock()
	defer rl.domainMux.RUnlock()

	currentCounter, ok := rl.perDomain[domain]
	if !ok {
		return counter{}, errDomainCounterNotFound
	}

	return currentCounter, nil
}

func (rl *RateLimiter) cleanup() {
	select {
	case t := <-rl.cleanupTimer.C:
		log.WithField("cleanup", t).Info("cleaning rate limiter")
		go func() {
			rl.domainMux.Lock()
			defer rl.domainMux.Unlock()
			for _, counter := range rl.perDomain {
				if rl.now().Sub(counter.lastSeen) > rl.domainWindow {
					counter.count -= rl.maxCountPerDomain
					if counter.count < 0 {
						counter.count = 0
					}
				}
			}
		}()
	default:

	}
}
