package source

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

func TestNewDomains(t *testing.T) {
	validCfg := config.GitLab{
		InternalServer:     "https://gitlab.com",
		APISecretKey:       []byte("abc"),
		ClientHTTPTimeout:  time.Second,
		JWTTokenExpiration: time.Second,
	}

	tests := []struct {
		name        string
		config      config.GitLab
		expectedErr string
	}{
		{
			name:   "gitlab_source_success",
			config: validCfg,
		},
		{
			name: "gitlab_source_no_url",
			config: func() config.GitLab {
				cfg := validCfg
				cfg.InternalServer = ""

				return cfg
			}(),
			expectedErr: "GitLab API URL or API secret has not been provided",
		},
		{
			name: "gitlab_source_no_secret",
			config: func() config.GitLab {
				cfg := validCfg
				cfg.APISecretKey = []byte{}

				return cfg
			}(),
			expectedErr: "GitLab API URL or API secret has not been provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains, err := NewDomains(&tt.config)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, domains.gitlab)
		})
	}
}

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
