package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

type mockSource struct {
	mock.Mock
}

func (m mockSource) GetDomain(name string) *domain.Domain {
	args := m.Called(name)

	return args.Get(0).(*domain.Domain)
}

func (m mockSource) HasDomain(name string) bool {
	args := m.Called(name)

	return args.Bool(0)
}

func TestSourceTransition(t *testing.T) {
	testDomain := newSourceDomains[0]

	t.Run("when requesting a test domain", func(t *testing.T) {
		newSource := new(mockSource)
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}).
			Once()
		newSource.On("HasDomain", testDomain).Return(true).Once()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		domains.GetDomain(testDomain)
		domains.HasDomain(testDomain)
	})

	t.Run("when requesting a non-test domain", func(t *testing.T) {
		newSource := new(mockSource)
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		assert.Nil(t, domains.GetDomain("some.test.io"))
		assert.False(t, domains.HasDomain("some.test.io"))
	})
}
