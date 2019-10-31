package gitlab

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubbedLookup struct {
	resolutions int
	domainChan  chan string
	failure     error
}

func (s *stubbedLookup) Resolve() (*Lookup, error) {
	s.resolutions++

	if s.failure != nil {
		return &Lookup{}, s.failure
	}

	return &Lookup{Domain: <-s.domainChan}, nil
}

func TestGetLookup(t *testing.T) {
	maxRetrievalInterval = 0

	t.Run("when item is not cached", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string, 1), failure: nil}
		resolver.domainChan <- "my.gitlab.com"

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, "my.gitlab.com", lookup.Domain)
		assert.Equal(t, 1, resolver.resolutions)
	})

	t.Run("when item is in short cache", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.store("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, "my.gitlab.com", lookup.Domain)
		assert.Equal(t, 0, resolver.resolutions)
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.store("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})
		cache.shortCache.Delete("my.gitlab.com")

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, "my.gitlab.com", lookup.Domain)
		assert.Equal(t, 0, resolver.resolutions)

		resolver.domainChan <- "my.gitlab.com"
		assert.Equal(t, 1, resolver.resolutions)
	})

	t.Run("when item in long cache is requested multiple times", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.store("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})
		cache.shortCache.Delete("my.gitlab.com")

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)
		cache.GetLookup("my.gitlab.com", resolver.Resolve)
		cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, "my.gitlab.com", lookup.Domain)
		assert.Equal(t, 0, resolver.resolutions)

		resolver.domainChan <- "my.gitlab.com"
		assert.Equal(t, 1, resolver.resolutions)
	})

	t.Run("when retrieval failed with an error", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{
			failure: errors.New("could not retrieve lookup"),
		}

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Nil(t, lookup)
		assert.Equal(t, 4, resolver.resolutions)
	})
}
