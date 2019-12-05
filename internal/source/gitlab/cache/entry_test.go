package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestIsUpToDateAndNeedsRefresh(t *testing.T) {
	t.Run("when is resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.response = &api.Lookup{}

		require.True(t, entry.IsUpToDate())
		require.False(t, entry.NeedsRefresh())
	})

	t.Run("when is resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.response = &api.Lookup{}
		entry.created = time.Now().Add(-time.Hour)

		require.False(t, entry.IsUpToDate())
		require.True(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")

		require.False(t, entry.IsUpToDate())
		require.False(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.created = time.Now().Add(-time.Hour)

		require.False(t, entry.IsUpToDate())
		require.False(t, entry.NeedsRefresh())
	})
}
