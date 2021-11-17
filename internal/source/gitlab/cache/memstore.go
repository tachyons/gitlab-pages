package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
)

type memstore struct {
	store                  *cache.Cache
	mux                    *sync.RWMutex
	entryRefreshTimeout    time.Duration
	entryExpirationTimeout time.Duration
}

func newMemStore(cc *config.Cache) Store {
	return &memstore{
		store:                  cache.New(cc.CacheExpiry, cc.CacheCleanupInterval),
		mux:                    &sync.RWMutex{},
		entryRefreshTimeout:    cc.EntryRefreshTimeout,
		entryExpirationTimeout: cc.CacheExpiry,
	}
}

// LoadOrCreate writes or retrieves a domain entry from the cache in a
// thread-safe way, trying to make this read-preferring RW locking.
func (m *memstore) LoadOrCreate(domain string) *Entry {
	m.mux.RLock()
	entry, exists := m.store.Get(domain)
	m.mux.RUnlock()

	if exists {
		return entry.(*Entry)
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	if entry, exists = m.store.Get(domain); exists {
		return entry.(*Entry)
	}

	newEntry := newCacheEntry(domain, m.entryRefreshTimeout, m.entryExpirationTimeout)
	m.store.SetDefault(domain, newEntry)

	return newEntry
}

func (m *memstore) ReplaceOrCreate(domain string, entry *Entry) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.store.Delete(domain)
	m.store.SetDefault(domain, entry)

	return entry
}
