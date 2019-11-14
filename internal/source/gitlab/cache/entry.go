package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	retrievalTimeout = 5 * time.Second
	shortCacheExpiry = 10 * time.Second
)

// Entry represents a cache object that can be retrieved asynchronously and
// holds a pointer to *Lookup when the domain lookup has been retrieved
// successfully
type Entry struct {
	domain    string
	created   time.Time
	retrieve  *sync.Once
	refresh   *sync.Once
	mux       *sync.RWMutex
	retrieved chan struct{}
	response  *Lookup
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
func (e *Entry) Lookup() *Lookup {
	e.mux.RLock()
	defer e.mux.RUnlock()

	return e.response
}

// Retrieve perform a blocking retrieval of the cache entry response.
func (e *Entry) Retrieve(ctx context.Context, client Resolver) *Lookup {
	newctx, cancelctx := context.WithTimeout(ctx, retrievalTimeout)
	defer cancelctx()

	e.retrieve.Do(func() { go e.retrieveWithClient(client) })

	select {
	case <-newctx.Done():
		return &Lookup{Status: 502, Error: errors.New("context done")}
	case <-e.retrieved:
		return e.Lookup()
	}

	// This should not happen
	return &Lookup{Status: 500, Error: errors.New("retrieval error")}
}

// Refresh will update the entry in the store only when it gets resolved.
func (e *Entry) Refresh(client Resolver, store Store) {
	e.refresh.Do(func() {
		go func() {
			entry := newCacheEntry(e.domain)

			entry.Retrieve(context.Background(), client)

			store.ReplaceOrCreate(e.domain, entry)
		}()
	})
}

func (e *Entry) retrieveWithClient(client Resolver) {
	retriever := Retriever{client: client, timeout: retrievalTimeout}

	e.setResponse(retriever.Retrieve(e.domain))
}

func (e *Entry) setResponse(lookup Lookup) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.response = &lookup
	close(e.retrieved)
}

func (e *Entry) isExpired() bool {
	return e.created.Add(shortCacheExpiry).Before(time.Now())
}

func (e *Entry) isResolved() bool {
	return e.response != nil
}
