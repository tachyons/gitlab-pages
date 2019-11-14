package cache

import (
	"context"
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
	memStore := &memstore{
		store: cache.New(expiration, time.Minute),
		mux:   &sync.Mutex{},
	}

	memStore.store.OnEvicted(memStore.OnEvicted)

	return memStore
}

func (m *memstore) LoadOrCreate(ctx context.Context, domain string) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	if entry, exists := m.store.Get(domain); exists {
		return entry.(*Entry)
	}

	entry := newCacheEntry(ctx, domain)
	m.store.SetDefault(domain, entry)

	return entry
}

func (m *memstore) ReplaceOrCreate(ctx context.Context, domain string, entry *Entry) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, exists := m.store.Get(domain); exists {
		m.store.Delete(domain) // delete manually to trigger onEvicted
	}

	m.store.SetDefault(domain, entry)

	return entry
}

func (m *memstore) OnEvicted(key string, value interface{}) {
	value.(*Entry).CancelRetrieval()
}
