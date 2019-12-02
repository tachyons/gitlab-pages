package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

func TestHasDomain(t *testing.T) {
	newSourceDomains = []string{"new-source-test.gitlab.io"}
	brokenSourceDomain = "pages-broken-poc.gitlab.io"

	t.Run("when requesting a test domain", func(t *testing.T) {
		testDomain := newSourceDomains[0]

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		domains.GetDomain(testDomain)
	})

	t.Run("when requesting a non-test domain", func(t *testing.T) {
		newSource := NewMockSource()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		domain, err := domains.GetDomain("domain.test.io")

		require.NoError(t, err)
		assert.Nil(t, domain)
	})

	t.Run("when requesting a broken test domain", func(t *testing.T) {
		newSource := NewMockSource()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		domain, err := domains.GetDomain("pages-broken-poc.gitlab.io")

		assert.Nil(t, domain)
		assert.EqualError(t, err, "broken test domain used")
	})
}
