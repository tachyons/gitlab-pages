package source

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

type sourceConfig struct {
	api    string
	secret string
	enable bool
}

func (c sourceConfig) GitlabServerURL() string {
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
func (c sourceConfig) GitlabEnableSourceAPI() bool {
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
	gitlabSourceConfig.Domains.Enabled = []string{"new-Source-test.gitlab.io"}
	gitlabSourceConfig.Domains.Broken = "pages-broken-poc.gitlab.io"

	t.Run("when requesting a test domain", func(t *testing.T) {
		testDomain := gitlabSourceConfig.Domains.Enabled[0]

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
		require.Nil(t, domain)
	})

	t.Run("when requesting a broken test domain", func(t *testing.T) {
		newSource := NewMockSource()
		defer newSource.AssertExpectations(t)

		domains := &Domains{
			disk:   disk.New(),
			gitlab: newSource,
		}

		domain, err := domains.GetDomain("pages-broken-poc.gitlab.io")

		require.Nil(t, domain)
		require.EqualError(t, err, "broken test domain used")
	})

	t.Run("when requesting a test domain in case of the Source not being fully configured", func(t *testing.T) {
		domains, err := NewDomains(sourceConfig{})
		require.NoError(t, err)

		domain, err := domains.GetDomain("new-Source-test.gitlab.io")

		require.Nil(t, domain)
		require.NoError(t, err)
	})

	t.Run("when requesting a serverless domain", func(t *testing.T) {
		testDomain := "func-aba1aabbccddeef2abaabbcc.serverless.gitlab.io"

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

func TestGetDomainWithIncrementalrolloutOfGitLabSource(t *testing.T) {
	// This will produce the following pseudo-random sequence: 5, 87, 68
	rand.Seed(42)

	// Generates FNV hash 4091421005, 4091421005 % 100 = 5
	domain05 := "test-domain-a.com"
	// Generates FNV 2643293380, 2643293380 % 100 = 80
	domain80 := "test-domain-b.com"

	diskSource := disk.New()

	gitlabSourceConfig.Domains.Rollout.Percentage = 80

	type testDomain struct {
		name   string
		source string
		times  int
	}

	tests := map[string]struct {
		stickiness string
		domains    []testDomain
	}{
		// domain05 should always use gitlab Source,
		// domain80 should use disk Source
		"default stickiness": {
			stickiness: "",
			domains: []testDomain{
				{name: domain05, source: "gitlab"},
				{name: domain80, source: "disk"},
				{name: domain05, source: "gitlab"},
			},
		},
		// Given that randSeed(42) will produce the following pseudo-random sequence:
		// {5, 87, 68} the first and third call for domain05 should use gitlab Source,
		// while the second one should use disk Source
		"no stickiness": {
			stickiness: "random",
			domains: []testDomain{
				{name: domain05, source: "gitlab"},
				{name: domain05, source: "disk"},
				{name: domain05, source: "gitlab"},
			}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gitlabSource := NewMockSource()
			for _, d := range tc.domains {
				if d.source == "gitlab" {
					gitlabSource.On("GetDomain", d.name).
						Return(&domain.Domain{Name: d.name}, nil).
						Once()
				}
			}
			defer gitlabSource.AssertExpectations(t)

			domains := &Domains{
				disk:   diskSource,
				gitlab: gitlabSource,
			}

			gitlabSourceConfig.Domains.Rollout.Stickiness = tc.stickiness

			for _, domain := range tc.domains {
				_, err := domains.GetDomain(domain.name)
				require.NoError(t, err)
			}
		})
	}
}