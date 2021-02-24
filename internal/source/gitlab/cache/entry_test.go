package cache

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestIsUpToDateAndNeedsRefresh(t *testing.T) {
	tests := []struct {
		name                string
		resolved            bool
		expired             bool
		expectedIsUpToDate  bool
		expectedNeedRefresh bool
	}{
		{
			name:                "resolved_and_not_expired",
			resolved:            true,
			expired:             false,
			expectedIsUpToDate:  true,
			expectedNeedRefresh: false,
		},
		{
			name:                "resolved_and_expired",
			resolved:            true,
			expired:             true,
			expectedIsUpToDate:  false,
			expectedNeedRefresh: true,
		},
		{
			name:                "not_resolved_and_not_expired",
			resolved:            false,
			expired:             false,
			expectedIsUpToDate:  false,
			expectedNeedRefresh: false,
		},
		{
			name:                "not_resolved_and_expired",
			resolved:            false,
			expired:             true,
			expectedIsUpToDate:  false,
			expectedNeedRefresh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := newCacheEntry("my.gitlab.com", testCacheConfig.EntryRefreshTimeout, testCacheConfig.CacheExpiry, nil)
			if tt.resolved {
				entry.response = &api.Lookup{}
			}
			if tt.expired {
				entry.created = time.Now().Add(-time.Hour)
			}

			require.Equal(t, tt.expectedIsUpToDate, entry.IsUpToDate())
			require.Equal(t, tt.expectedNeedRefresh, entry.NeedsRefresh())
		})
	}
}

func TestEntryRefresh(t *testing.T) {
	client := &lookupMock{
		successCount: 1,
		responses: map[string]api.Lookup{
			"test.gitlab.io": api.Lookup{
				Name: "test.gitlab.io",
				Domain: &api.VirtualDomain{
					LookupPaths: nil,
				},
			},
			"error.gitlab.io": api.Lookup{
				Name:  "error.gitlab.io",
				Error: errors.New("something went wrong"),
			},
		},
	}
	cc := &cacheConfig{
		cacheExpiry:          100 * time.Millisecond,
		entryRefreshTimeout:  time.Millisecond,
		retrievalTimeout:     50 * time.Millisecond,
		maxRetrievalInterval: time.Millisecond,
		maxRetrievalRetries:  1,
	}

	store := newMemStore(client, cc)

	t.Run("entry is the same after refreshed lookup has error", func(t *testing.T) {
		entry := newCacheEntry("test.gitlab.io", cc.entryRefreshTimeout, cc.cacheExpiry, store.(*memstore).retriever)

		ctx, cancel := context.WithTimeout(context.Background(), cc.retrievalTimeout)
		defer cancel()

		lookup := entry.Retrieve(ctx)
		require.NoError(t, lookup.Error)

		require.Eventually(t, entry.NeedsRefresh, 100*time.Millisecond, time.Millisecond, "entry should need refresh")

		entry.refreshFunc(store)

		require.True(t, client.failed, "refresh should have failed")

		storedEntry := loadEntry(t, "test.gitlab.io", store)

		require.NoError(t, storedEntry.Lookup().Error, "resolving failed but lookup should still be valid")
		require.Equal(t, storedEntry.created.UnixNano(), entry.created.UnixNano(), "refreshed entry should be the same")
		require.Equal(t, storedEntry.Lookup(), entry.Lookup(), "lookup should be the same")
	})

	t.Run("entry is different after it expired and calling refresh on it", func(t *testing.T) {
		client.failed = false

		entry := newCacheEntry("error.gitlab.io", cc.entryRefreshTimeout, cc.cacheExpiry, store.(*memstore).retriever)

		ctx, cancel := context.WithTimeout(context.Background(), cc.retrievalTimeout)
		defer cancel()

		lookup := entry.Retrieve(ctx)
		require.Error(t, lookup.Error)
		require.Eventually(t, entry.NeedsRefresh, 100*time.Millisecond, time.Millisecond, "entry should need refresh")

		// wait for entry to expire
		time.Sleep(cc.cacheExpiry)
		// refreshing the entry after it has expired should create a completely new one
		entry.refreshFunc(store)

		require.True(t, client.failed, "refresh should have failed")

		storedEntry := loadEntry(t, "error.gitlab.io", store)
		require.NotEqual(t, storedEntry, entry, "stored entry should be different")
		require.Greater(t, storedEntry.created.UnixNano(), entry.created.UnixNano(), "")
	})
}

func loadEntry(t *testing.T, domain string, store Store) *Entry {
	t.Helper()

	i, exists := store.(*memstore).store.Get(domain)
	require.True(t, exists)

	return i.(*Entry)
}

type lookupMock struct {
	currentCount int
	successCount int
	failed       bool
	responses    map[string]api.Lookup
}

func (lm *lookupMock) GetLookup(ctx context.Context, domainName string) api.Lookup {
	lookup := api.Lookup{
		Name: domainName,
	}

	lookup, ok := lm.responses[domainName]
	if !ok {
		lookup.Error = domain.ErrDomainDoesNotExist
		return lookup
	}

	// return error after lm.successCount
	lm.currentCount++
	if lm.currentCount > lm.successCount {
		lm.currentCount = 0
		lm.failed = true

		lookup.Error = http.ErrServerClosed
	}

	return lookup
}

func (lm *lookupMock) Status() error {
	return nil
}
