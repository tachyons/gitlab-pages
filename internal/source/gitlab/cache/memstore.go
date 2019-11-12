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

func newMemStore() Store {
	memStore := &memstore{
		// TODO onEvicted cancel context
		store: cache.New(10*time.Minute, time.Minute),
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

func (m *memstore) ReplaceOrCreate(ctx context.Context, domain string) *Entry {
	m.mux.Lock()
	defer m.mux.Unlock()

	entry := newCacheEntry(ctx, domain)

	if _, exists := m.store.Get(domain); exists {
		m.store.Delete(domain) // delete manually to trigger onEvicted
	}

	m.store.SetDefault(domain, entry)

	return entry
}

func (m *memstore) OnEvicted(key string, value interface{}) {
	value.(*Entry).CancelContexts()
}
