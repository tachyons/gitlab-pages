package gitlab

import (
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

	return &Lookup{Domain: <-s.domainChan}, s.failure
}

func TestGetLookup(t *testing.T) {
	t.Run("when item is not cached", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string, 1), failure: nil}
		resolver.domainChan <- "my.gitlab.com"

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, lookup.Domain, "my.gitlab.com")
		assert.Equal(t, resolver.resolutions, 1)
	})

	t.Run("when item is in short cache", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.storeEntry("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, lookup.Domain, "my.gitlab.com")
		assert.Equal(t, resolver.resolutions, 0)
	})

	t.Run("when item is in long cache only", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.storeEntry("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})
		cache.shortCache.Delete("my.gitlab.com")

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, lookup.Domain, "my.gitlab.com")
		assert.Equal(t, resolver.resolutions, 0)

		resolver.domainChan <- "my.gitlab.com"
		assert.Equal(t, resolver.resolutions, 1)
	})

	t.Run("when item in long cache is requested multiple times", func(t *testing.T) {
		cache := NewCache()
		resolver := stubbedLookup{domainChan: make(chan string), failure: nil}
		cache.storeEntry("my.gitlab.com", &Lookup{Domain: "my.gitlab.com"})
		cache.shortCache.Delete("my.gitlab.com")

		lookup := cache.GetLookup("my.gitlab.com", resolver.Resolve)
		cache.GetLookup("my.gitlab.com", resolver.Resolve)
		cache.GetLookup("my.gitlab.com", resolver.Resolve)

		assert.Equal(t, lookup.Domain, "my.gitlab.com")
		assert.Equal(t, resolver.resolutions, 0)

		resolver.domainChan <- "my.gitlab.com"
		assert.Equal(t, resolver.resolutions, 1)
	})
}
