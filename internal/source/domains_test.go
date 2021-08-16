package source

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

func TestGetDomain(t *testing.T) {
	t.Run("when requesting an existing domain for gitlab source", func(t *testing.T) {
		testDomain := "new-source-test.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource)

		domain, err := domains.GetDomain(context.Background(), testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("when requesting a domain that doesn't exist for gitlab source", func(t *testing.T) {
		newSource := NewMockSource()
		newSource.On("GetDomain", "does-not-exist.test.io").
			Return(nil, nil).
			Once()

		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource)

		domain, err := domains.GetDomain(context.Background(), "does-not-exist.test.io")
		require.NoError(t, err)
		require.Nil(t, domain)
	})
}

func newTestDomains(t *testing.T, gitlabSource *MockSource) *Domains {
	t.Helper()

	return &Domains{
		gitlab: gitlabSource,
	}
}
