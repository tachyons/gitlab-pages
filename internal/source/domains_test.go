package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

type sourceConfig struct {
	api    string
	secret string
}

func (c sourceConfig) GitlabServerURL() string {
	return c.api
}

func (c sourceConfig) GitlabAPISecret() []byte {
	return []byte(c.secret)
}

func TestDomainSources(t *testing.T) {
	t.Run("when GitLab API URL has been provided", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{api: "https://gitlab.com"})
		require.Nil(t, err)

		assert.NotNil(t, domains.gitlab)
		assert.NotNil(t, domains.disk)
	})

	t.Run("when GitLab API has not been provided", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{})
		require.Nil(t, err)

		assert.Nil(t, domains.gitlab)
		assert.NotNil(t, domains.disk)
	})
}

func TestGetDomain(t *testing.T) {
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

	t.Run("when requesting a test domain in case of the source not being fully configured", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{})
		require.NoError(t, err)

		domain, err := domains.GetDomain("new-source-test.gitlab.io")

		assert.Nil(t, domain)
		assert.NoError(t, err)
	})
}
