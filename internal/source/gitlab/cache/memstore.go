package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type memstore struct {
	store               *cache.Cache
	mux                 *sync.RWMutex
	retriever           *Retriever
	entryRefreshTimeout time.Duration
}

func newMemStore(client api.Client, cc *config.Cache) Store {
	retriever := NewRetriever(client, cc.RetrievalTimeout, cc.MaxRetrievalInterval, cc.MaxRetrievalRetries)

	return &memstore{
		store:               cache.New(cc.CacheExpiry, cc.CacheCleanupInterval),
		mux:                 &sync.RWMutex{},
		retriever:           retriever,
		entryRefreshTimeout: cc.EntryRefreshTimeout,
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

	newEntry := newCacheEntry(domain, m.entryRefreshTimeout, m.retriever)
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
