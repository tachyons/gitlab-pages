package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

func TestNewDomains(t *testing.T) {
	validCfg := config.GitLab{
		InternalServer:     "https://gitlab.com",
		APISecretKey:       []byte("abc"),
		ClientHTTPTimeout:  time.Second,
		JWTTokenExpiration: time.Second,
	}

	tests := []struct {
		name            string
		source          string
		config          config.GitLab
		expectedErr     string
		expectGitlabNil bool
		expectDiskNil   bool
	}{
		{
			name:        "no_source_config",
			source:      "",
			expectedErr: "invalid option for -domain-config-source: \"\"",
		},
		{
			name:        "invalid_source_config",
			source:      "invalid",
			expectedErr: "invalid option for -domain-config-source: \"invalid\"",
		},
		{
			name:            "disk_source",
			source:          "disk",
			expectGitlabNil: true,
			expectDiskNil:   false,
		},
		{
			name:            "auto_without_api_config",
			source:          "auto",
			expectGitlabNil: true,
			expectDiskNil:   false,
		},
		{
			name:            "auto_with_api_config",
			source:          "auto",
			config:          validCfg,
			expectGitlabNil: false,
			expectDiskNil:   false,
		},
		{
			name:          "gitlab_source_success",
			source:        "gitlab",
			config:        validCfg,
			expectDiskNil: true,
		},
		{
			name:   "gitlab_source_no_url",
			source: "gitlab",
			config: func() config.GitLab {
				cfg := validCfg
				cfg.InternalServer = ""

				return cfg
			}(),
			expectedErr: "GitLab API URL or API secret has not been provided",
		},
		{
			name:   "gitlab_source_no_secret",
			source: "gitlab",
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
			domains, err := NewDomains(tt.source, &tt.config)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.expectGitlabNil, domains.gitlab == nil, "mismatch gitlab nil")
			require.Equal(t, tt.expectDiskNil, domains.disk == nil, "mismatch disk nil")
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

		domains := newTestDomains(t, newSource, sourceGitlab)

		domain, err := domains.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("when requesting an existing domain for auto source", func(t *testing.T) {
		testDomain := "new-source-test.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		newSource.On("IsReady").Return(true).Once()
		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource, sourceAuto)

		domain, err := domains.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("when requesting a domain that doesn't exist for gitlab source", func(t *testing.T) {
		newSource := NewMockSource()
		newSource.On("GetDomain", "does-not-exist.test.io").
			Return(nil, nil).
			Once()

		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource, sourceGitlab)

		domain, err := domains.GetDomain("does-not-exist.test.io")
		require.NoError(t, err)
		require.Nil(t, domain)
	})

	t.Run("when requesting a serverless domain", func(t *testing.T) {
		testDomain := "func-aba1aabbccddeef2abaabbcc.serverless.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()

		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource, sourceGitlab)

		domain, err := domains.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})
}

func TestIsServerlessDomain(t *testing.T) {
	t.Run("when a domain is serverless domain", func(t *testing.T) {
		require.True(t, IsServerlessDomain("some-function-aba1aabbccddeef2abaabbcc.serverless.gitlab.io"))
	})

	t.Run("when a domain is serverless domain with environment", func(t *testing.T) {
		require.True(t, IsServerlessDomain("some-function-aba1aabbccddeef2abaabbcc-testing.serverless.gitlab.io"))
	})

	t.Run("when a domain is not a serverless domain", func(t *testing.T) {
		require.False(t, IsServerlessDomain("somedomain.gitlab.io"))
	})
}

func newTestDomains(t *testing.T, gitlabSource *MockSource, config configSource) *Domains {
	t.Helper()

	return &Domains{
		configSource: config,
		gitlab:       gitlabSource,
		disk:         disk.New(),
	}
}
