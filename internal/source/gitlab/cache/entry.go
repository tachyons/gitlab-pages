package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Entry represents a cache object that can be retrieved asynchronously and
// holds a pointer to *api.Lookup when the domain lookup has been retrieved
// successfully
type Entry struct {
	domain         string
	created        time.Time
	retrieve       *sync.Once
	refresh        *sync.Once
	mux            *sync.RWMutex
	retrieved      chan struct{}
	response       *api.Lookup
	refreshTimeout time.Duration
	maxDuration    time.Duration
	retriever      *Retriever
}

func newCacheEntry(domain string, refreshTimeout, maxDuration time.Duration, retriever *Retriever) *Entry {
	return &Entry{
		domain:         domain,
		created:        time.Now(),
		retrieve:       &sync.Once{},
		refresh:        &sync.Once{},
		mux:            &sync.RWMutex{},
		retrieved:      make(chan struct{}),
		refreshTimeout: refreshTimeout,
		maxDuration:    maxDuration,
		retriever:      retriever,
	}
}

// IsUpToDate returns true if the entry has been resolved correctly and has not
// expired yet. False otherwise.
func (e *Entry) IsUpToDate() bool {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.isResolved() && !e.isExpired()
}

// NeedsRefresh return true if the entry has been resolved correctly but it has
// expired since then.
func (e *Entry) NeedsRefresh() bool {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.isResolved() && e.isExpired()
}

// Lookup returns a retriever Lookup response.
func (e *Entry) Lookup() *api.Lookup {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.response
}

// Retrieve perform a blocking retrieval of the cache entry response.
func (e *Entry) Retrieve(ctx context.Context) (lookup *api.Lookup) {
	// We run the code within an additional func() to run both `e.setResponse`
	// and `e.retrieve.Retrieve` asynchronously.
	e.retrieve.Do(func() { go func() { e.setResponse(e.retriever.Retrieve(e.domain)) }() })

	select {
	case <-ctx.Done():
		lookup = &api.Lookup{Name: e.domain, Error: errors.New("context done")}
	case <-e.retrieved:
		lookup = e.Lookup()
	}

	return lookup
}

// Refresh will update the entry in the store only when it gets resolved.
func (e *Entry) Refresh(store Store) {
	e.refresh.Do(func() {
		go e.refreshFunc(store)
	})
}

func (e *Entry) refreshFunc(store Store) {
	entry := newCacheEntry(e.domain, e.refreshTimeout, e.maxDuration, e.retriever)

	entry.Retrieve(context.Background())
	if e.reuseEntry(entry) {
		entry.response = e.response
		entry.created = e.created
	}

	store.ReplaceOrCreate(e.domain, entry)
}

func (e *Entry) setResponse(lookup api.Lookup) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.response = &lookup
	close(e.retrieved)
}

func (e *Entry) isExpired() bool {
	return time.Since(e.created) > e.refreshTimeout
}

func (e *Entry) isResolved() bool {
	return e.response != nil
}

func (e *Entry) maxTimeInCache() bool {
	return time.Since(e.created) > e.maxDuration
}

// reuseEntry as the refreshed entry when there is an error after resolving the lookup again
// and is different to domain.ErrDomainDoesNotExist. This is an edge case to prevent serving
// a page right after being deleted.
// It should only be refreshed as long as it hasn't passed e.maxDuration.
// See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/281.
func (e *Entry) reuseEntry(entry *Entry) bool {
	return entry.response != nil &&
		entry.response.Error != nil &&
		!errors.Is(entry.response.Error, domain.ErrDomainDoesNotExist) &&
		!e.maxTimeInCache()
}
