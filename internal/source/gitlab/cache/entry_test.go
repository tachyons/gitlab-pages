package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestIsUpToDateAndNeedsRefresh(t *testing.T) {
	t.Run("when is resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.response = &api.Lookup{}

		assert.True(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})

	t.Run("when is resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.response = &api.Lookup{}
		entry.created = time.Now().Add(-time.Hour)

		assert.False(t, entry.IsUpToDate())
		assert.True(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")

		assert.False(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry("my.gitlab.com")
		entry.created = time.Now().Add(-time.Hour)

		assert.False(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})
}
