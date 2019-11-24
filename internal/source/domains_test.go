package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

type mockSource struct {
	mock.Mock
}

func (m *mockSource) GetDomain(name string) (*domain.Domain, error) {
	args := m.Called(name)

	return args.Get(0).(*domain.Domain), args.Error(1)
}

func TestHasDomain(t *testing.T) {
	testDomain := newSourceDomains[0]

	t.Run("when requesting a test domain", func(t *testing.T) {
		newSource := new(mockSource)
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
		newSource := new(mockSource)
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
		newSource := new(mockSource)
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
