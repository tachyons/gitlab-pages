package cache

import (
	"sync"
	"time"

	cache "github.com/patrickmn/go-cache"
)

type memstore struct {
	store *cache.Cache
	mux   *sync.Mutex
}

var expiration = 10 * time.Minute

func newMemStore() Store {
	return &memstore{
		store: cache.New(expiration, time.Minute),
		mux:   &sync.Mutex{},
	}
}

func (m *memstore) LoadOrCreate(domain string) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	if entry, exists := m.store.Get(domain); exists {
		return entry.(*Entry)
	}

	entry := newCacheEntry(domain)
	m.store.SetDefault(domain, entry)

	return entry
}

func (m *memstore) ReplaceOrCreate(domain string, entry *Entry) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, exists := m.store.Get(domain); exists {
		m.store.Delete(domain)
	}

	m.store.SetDefault(domain, entry)

	return entry
}
