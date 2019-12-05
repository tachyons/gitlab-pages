package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Entry represents a cache object that can be retrieved asynchronously and
// holds a pointer to *api.Lookup when the domain lookup has been retrieved
// successfully
type Entry struct {
	domain    string
	created   time.Time
	retrieve  *sync.Once
	refresh   *sync.Once
	mux       *sync.RWMutex
	retrieved chan struct{}
	response  *api.Lookup
}

func newCacheEntry(domain string) *Entry {
	return &Entry{
		domain:    domain,
		created:   time.Now(),
		retrieve:  &sync.Once{},
		refresh:   &sync.Once{},
		mux:       &sync.RWMutex{},
		retrieved: make(chan struct{}),
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
func (e *Entry) Retrieve(ctx context.Context, client api.Client) (lookup *api.Lookup) {
	newctx, cancelctx := context.WithTimeout(ctx, retrievalTimeout)
	defer cancelctx()

	e.retrieve.Do(func() { go e.retrieveWithClient(client) })

	select {
	case <-newctx.Done():
		lookup = &api.Lookup{Name: e.domain, Error: errors.New("context done")}
	case <-e.retrieved:
		lookup = e.Lookup()
	}

	return lookup
}

// Refresh will update the entry in the store only when it gets resolved.
func (e *Entry) Refresh(client api.Client, store Store) {
	e.refresh.Do(func() {
		go func() {
			entry := newCacheEntry(e.domain)

			entry.Retrieve(context.Background(), client)

			store.ReplaceOrCreate(e.domain, entry)
		}()
	})
}

func (e *Entry) retrieveWithClient(client api.Client) {
	retriever := Retriever{client: client}

	e.setResponse(retriever.Retrieve(e.domain))
}

func (e *Entry) setResponse(lookup api.Lookup) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.response = &lookup
	close(e.retrieved)
}

func (e *Entry) isExpired() bool {
	return time.Since(e.created) > shortCacheExpiry
}

func (e *Entry) isResolved() bool {
	return e.response != nil
}
