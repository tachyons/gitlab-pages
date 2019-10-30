package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubbedLookup struct {
	resolutions int
}

func (s *stubbedLookup) Resolve() *Lookup {
	s.resolutions++

	return &Lookup{Domain: "my.gitlab.com"}
}

func TestGetLookup(t *testing.T) {
	t.Run("when item is not cached", func(t *testing.T) {
		cache := NewCache()
		resolver := new(stubbedLookup)

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, lookup.Domain, "my.gitlab.com")
		assert.Equal(t, resolver.resolutions, 1)
	})
}
