package source

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type sourceConfig struct {
	api     string
	secret  string
	disable bool
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
func (c sourceConfig) GitlabDisableAPIConfigurationSource() bool {
	return c.disable
}

func TestDomainSources(t *testing.T) {
	// TODO refactor test when disk source is removed https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
	tests := []struct {
		name         string
		config       sourceConfig
		mock         bool
		status       int
		expectGitlab bool
		expectDisk   bool
	}{
		{
			name:         "gitlab_source_on_success",
			config:       sourceConfig{api: "http://localhost", secret: "abc"},
			mock:         true,
			status:       http.StatusNoContent,
			expectGitlab: true,
		},
		{
			name:       "disk_source_on_unauthorized",
			config:     sourceConfig{api: "http://localhost", secret: "abc"},
			mock:       true,
			status:     http.StatusUnauthorized,
			expectDisk: true,
		},
		{
			name:       "disk_source_on_api_error",
			config:     sourceConfig{api: "http://localhost", secret: "abc"},
			mock:       true,
			status:     http.StatusServiceUnavailable,
			expectDisk: true,
		},
		{
			name:       "disk_source_on_disabled_api_source",
			config:     sourceConfig{api: "http://localhost", secret: "abc", disable: true},
			expectDisk: true,
		},
		{
			name:       "disk_source_on_incomplete_config",
			config:     sourceConfig{api: "", secret: "abc"},
			expectDisk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock {
				m := http.NewServeMux()
				m.HandleFunc("/api/v4/internal/pages/status", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
				})

				mockServer := httptest.NewServer(m)
				defer mockServer.Close()

				tt.config.api = mockServer.URL
			}

			domains, err := NewDomains(tt.config)
			require.NoError(t, err)

			require.Equal(t, tt.expectGitlab, domains.gitlab != nil)
			require.Equal(t, tt.expectDisk, domains.disk != nil)
		})
	}

}

func TestGetDomain(t *testing.T) {
	t.Run("when requesting a domain that exists", func(t *testing.T) {
		testDomain := "new-source-test.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			gitlab: newSource,
		}

		d, err := domains.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Equal(t, d.Name, testDomain)
	})

	t.Run("when requesting a domain that doesn't exist", func(t *testing.T) {
		newSource := NewMockSource()
		newSource.On("GetDomain", mock.Anything).Return(nil, nil)
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			gitlab: newSource,
		}

		d, err := domains.GetDomain("domain.test.io")
		require.NoError(t, err)
		require.Nil(t, d)
	})
}
