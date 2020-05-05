package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type sourceConfig struct {
	api    string
	secret string
	enable bool
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
	return c.enable
}

func TestDomainSources(t *testing.T) {
	t.Run("when GitLab API URL has been provided but cannot authenticate", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{api: "https://gitlab.com", secret: "abc", enable: true})
		require.NoError(t, err)

		require.Nil(t, domains.gitlab)
		require.NotNil(t, domains.disk)
	})

	t.Run("when GitLab API has not been provided", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{})
		require.NoError(t, err)

		require.Nil(t, domains.gitlab)
		require.NotNil(t, domains.disk)
	})
}

func TestGetDomain(t *testing.T) {
	t.Run("when requesting a test domain", func(t *testing.T) {
		testDomain := "new-source-test.gitlab.io"

		newSource := NewMockSource()
		newSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			gitlab: newSource,
		}

		domains.GetDomain(testDomain)
	})

	t.Run("when requesting a non-test domain", func(t *testing.T) {
		newSource := NewMockSource()
		newSource.On("GetDomain", mock.Anything).Return(nil, nil)
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			gitlab: newSource,
		}

		domain, err := domains.GetDomain("domain.test.io")

		require.NoError(t, err)
		require.Nil(t, domain)
	})

	t.Run("when requesting a test domain in case of the Source not being fully configured", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{})
		require.NoError(t, err)

		domain, err := domains.GetDomain("new-source-test.gitlab.io")

		require.Nil(t, domain)
		require.NoError(t, err)
	})

}
