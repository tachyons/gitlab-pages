package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

type sourceConfig struct {
	api          string
	secret       string
	domainSource string
}

func (c sourceConfig) InternalGitLabServerURL() string {
	return c.api
}

func (c sourceConfig) GitlabAPISecret() []byte {
	return []byte(c.secret)
}
func (c sourceConfig) GitlabClientConnectionTimeout() time.Duration {
	return 10 * time.Second
}

func (c sourceConfig) GitlabJWTTokenExpiry() time.Duration {
	return 30 * time.Second
}

func (c sourceConfig) DomainConfigSource() string {
	return c.domainSource
}

func TestDomainSources(t *testing.T) {
	tests := []struct {
		name            string
		sourceConfig    sourceConfig
		wantErr         bool
		expectGitlabNil bool
	}{
		{
			name:            "no_source_config",
			sourceConfig:    sourceConfig{},
			wantErr:         false,
			expectGitlabNil: true,
		},
		{
			name:            "disk_source",
			sourceConfig:    sourceConfig{domainSource: "disk"},
			wantErr:         false,
			expectGitlabNil: true,
		},
		{
			name:         "gitlab_source_success",
			sourceConfig: sourceConfig{api: "https://gitlab.com", secret: "abc", domainSource: "gitlab"},
			wantErr:      false,
		},
		{
			name:         "gitlab_source_no_url",
			sourceConfig: sourceConfig{api: "", secret: "abc", domainSource: "gitlab"},
			wantErr:      true,
		},
		{
			name:         "gitlab_source_no_secret",
			sourceConfig: sourceConfig{api: "https://gitlab.com", secret: "", domainSource: "gitlab"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains, err := NewDomains(tt.sourceConfig)
			if tt.wantErr {
				require.EqualError(t, err, "missing -internal-gitlab-server and/or -api-secret-key")
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.expectGitlabNil, domains.gitlab == nil)
			require.NotNil(t, domains.disk)
		})
	}
}

func TestGetDomain(t *testing.T) {
	t.Run("when requesting an existing domain", func(t *testing.T) {
		testDomain := "new-source-test.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		newSource.On("IsReady").Return(true).Once()
		defer newSource.AssertExpectations(t)

		domains := newTestDomains(t, newSource, sourceGitlab)

		domain, err := domains.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("when requesting a domain that doesn't exist", func(t *testing.T) {
		newSource := NewMockSource()
		newSource.On("GetDomain", "does-not-exist.test.io").
			Return(nil, nil).
			Once()
		newSource.On("IsReady").Return(true).Once()

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
