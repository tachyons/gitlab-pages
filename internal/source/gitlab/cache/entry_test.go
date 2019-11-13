package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsUpToDateAndNeedsRefresh(t *testing.T) {
	t.Run("when is resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry(context.Background(), "my.gitlab.com")
		entry.response = &Lookup{}

		assert.True(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})

	t.Run("when is resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry(context.Background(), "my.gitlab.com")
		entry.response = &Lookup{}
		entry.created = time.Now().Add(-time.Hour)

		assert.False(t, entry.IsUpToDate())
		assert.True(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and not expired", func(t *testing.T) {
		entry := newCacheEntry(context.Background(), "my.gitlab.com")

		assert.False(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})

	t.Run("when is not resolved and is expired", func(t *testing.T) {
		entry := newCacheEntry(context.Background(), "my.gitlab.com")
		entry.created = time.Now().Add(-time.Hour)

		assert.False(t, entry.IsUpToDate())
		assert.False(t, entry.NeedsRefresh())
	})
}
