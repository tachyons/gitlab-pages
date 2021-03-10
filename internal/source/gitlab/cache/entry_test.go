package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
			entry := newCacheEntry("my.gitlab.com", testCacheConfig.EntryRefreshTimeout, nil)
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
