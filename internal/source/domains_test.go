package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
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

func TestNewDomains(t *testing.T) {
	tests := []struct {
		name         string
		sourceConfig sourceConfig
		expectedErr  string
		expectedType interface{}
	}{
		{
			name:         "no_source_config",
			sourceConfig: sourceConfig{},
			expectedErr:  "invalid option for -domain-config-source: \"\"",
		},
		{
			name:         "invalid_source_config",
			sourceConfig: sourceConfig{domainSource: "invalid"},
			expectedErr:  "invalid option for -domain-config-source: \"invalid\"",
		},
		{
			name:         "disk_source",
			sourceConfig: sourceConfig{domainSource: "disk"},
			expectedType: &diskServerlessSource{},
		},
		{
			name:         "auto_without_api_config",
			sourceConfig: sourceConfig{domainSource: "auto"},
			expectedType: &autoSource{},
		},
		{
			name:         "auto_with_api_config",
			sourceConfig: sourceConfig{api: "https://gitlab.com", secret: "abc", domainSource: "auto"},
			expectedType: &autoSource{},
		},
		{
			name:         "gitlab_source_success",
			sourceConfig: sourceConfig{api: "https://gitlab.com", secret: "abc", domainSource: "gitlab"},
			expectedType: &gitlab.Gitlab{},
		},
		{
			name:         "gitlab_source_no_url",
			sourceConfig: sourceConfig{api: "", secret: "abc", domainSource: "gitlab"},
			expectedErr:  "GitLab API URL or API secret has not been provided",
		},
		{
			name:         "gitlab_source_no_secret",
			sourceConfig: sourceConfig{api: "https://gitlab.com", secret: "", domainSource: "gitlab"},
			expectedErr:  "GitLab API URL or API secret has not been provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains, err := NewDomains(tt.sourceConfig)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tt.expectedType, domains)
		})
	}
}

func TestNotAvailableSource(t *testing.T) {
	source := &notAvailableSource{}

	domain, err := source.GetDomain("example.com")
	require.Nil(t, domain)
	require.Error(t, err)

	require.False(t, source.IsReady())
}

func TestNewAutoSource(t *testing.T) {
	t.Run("with valid API config sets up both gitlab and disk source", func(t *testing.T) {
		source := newAutoSource(sourceConfig{api: "https://gitlab.com", secret: "abc"})

		require.IsType(t, &gitlab.Gitlab{}, source.gitlab)
		require.IsType(t, &disk.Disk{}, source.disk)
	})

	t.Run("with empty API config sets up only disk source", func(t *testing.T) {
		source := newAutoSource(sourceConfig{})

		require.IsType(t, notAvailableSource{}, source.gitlab)
		require.IsType(t, &disk.Disk{}, source.disk)
	})

	t.Run("with invalid API config sets up only disk source", func(t *testing.T) {
		source := newAutoSource(sourceConfig{api: "https://gitlab.com"})

		require.IsType(t, notAvailableSource{}, source.gitlab)
		require.IsType(t, &disk.Disk{}, source.disk)
	})
}

func TestAutoSourceGetDomain(t *testing.T) {
	t.Run("uses gitlab source for serverless domains regardless of gitlab source being ready", func(t *testing.T) {
		testDomain := "func-aba1aabbccddeef2abaabbcc.serverless.gitlab.io"

		gitlabSource := NewMockSource()
		gitlabSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()

		defer gitlabSource.AssertExpectations(t)

		source := &autoSource{gitlab: gitlabSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("uses gitlab source if it's ready", func(t *testing.T) {
		testDomain := "example.com"

		gitlabSource := NewMockSource()
		gitlabSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()
		gitlabSource.On("IsReady").Return(true).Once()

		defer gitlabSource.AssertExpectations(t)

		source := &autoSource{gitlab: gitlabSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("returns nil if gitlab source couldn't find it", func(t *testing.T) {
		testDomain := "example.com"

		gitlabSource := NewMockSource()
		gitlabSource.On("GetDomain", testDomain).
			Return(nil, nil).
			Once()
		gitlabSource.On("IsReady").Return(true).Once()

		defer gitlabSource.AssertExpectations(t)

		source := &autoSource{gitlab: gitlabSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.Nil(t, domain)
	})

	t.Run("uses disk source if gitlab source isn't ready", func(t *testing.T) {
		testDomain := "example.com"

		gitlabSource := NewMockSource()
		gitlabSource.On("IsReady").Return(false).Once()

		diskSource := NewMockSource()
		diskSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()

		defer gitlabSource.AssertExpectations(t)
		defer diskSource.AssertExpectations(t)

		source := &autoSource{gitlab: gitlabSource, disk: diskSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})
}

func TestAutoSourceIsReady(t *testing.T) {
	tests := []struct {
		name              string
		gitlabSourceReady bool
		diskSourceReady   bool
		expectedResult    bool
	}{
		{
			name:              "when both sources aren't ready",
			gitlabSourceReady: false,
			diskSourceReady:   false,
			expectedResult:    false,
		},
		{
			name:              "when gitlab source is ready",
			gitlabSourceReady: true,
			diskSourceReady:   false,
			expectedResult:    true,
		},
		{
			name:              "when disk source is ready",
			gitlabSourceReady: false,
			diskSourceReady:   true,
			expectedResult:    true,
		},
		{
			name:              "when both sources are ready",
			gitlabSourceReady: false,
			diskSourceReady:   true,
			expectedResult:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitlabSource := NewMockSource()
			gitlabSource.On("IsReady").Return(tt.gitlabSourceReady).Once()

			diskSource := NewMockSource()
			diskSource.On("IsReady").Return(tt.diskSourceReady).Once()

			source := &autoSource{gitlab: gitlabSource, disk: diskSource}

			require.Equal(t, tt.expectedResult, source.IsReady())
		})
	}
}

func TestAutoSourceRead(t *testing.T) {
	diskSource := NewMockSource()
	diskSource.On("Read", "gitlab.io").Once()

	defer diskSource.AssertExpectations(t)

	source := &autoSource{disk: diskSource}

	source.Read("gitlab.io")
}

func TestNewDiskServerlessSource(t *testing.T) {
	t.Run("with valid API config sets up both gitlab and disk source", func(t *testing.T) {
		source := newDiskServerlessSource(sourceConfig{api: "https://gitlab.com", secret: "abc"})

		require.IsType(t, &gitlab.Gitlab{}, source.serverless)
		require.IsType(t, &disk.Disk{}, source.disk)
	})

	t.Run("with empty API config sets up only disk source", func(t *testing.T) {
		source := newDiskServerlessSource(sourceConfig{})

		require.IsType(t, notAvailableSource{}, source.serverless)
		require.IsType(t, &disk.Disk{}, source.disk)
	})

	t.Run("with invalid API config sets up only disk source", func(t *testing.T) {
		source := newDiskServerlessSource(sourceConfig{api: "https://gitlab.com"})

		require.IsType(t, notAvailableSource{}, source.serverless)
		require.IsType(t, &disk.Disk{}, source.disk)
	})
}

func TestDiskServerlessSourceGetDomain(t *testing.T) {
	t.Run("uses gitlab source for serverless domains regardless of gitlab source being ready", func(t *testing.T) {
		testDomain := "func-aba1aabbccddeef2abaabbcc.serverless.gitlab.io"

		gitlabSource := NewMockSource()
		gitlabSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()

		defer gitlabSource.AssertExpectations(t)

		source := &diskServerlessSource{serverless: gitlabSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})

	t.Run("uses disk source for other domains", func(t *testing.T) {
		testDomain := "example.com"

		diskSource := NewMockSource()
		diskSource.On("GetDomain", testDomain).
			Return(&domain.Domain{Name: testDomain}, nil).
			Once()

		defer diskSource.AssertExpectations(t)

		source := &diskServerlessSource{disk: diskSource}

		domain, err := source.GetDomain(testDomain)
		require.NoError(t, err)
		require.NotNil(t, domain)
	})
}

func TestDiskServerlessSourceIsReady(t *testing.T) {
	gitlabSource := NewMockSource()
	gitlabSource.On("IsReady").Return(false).Once()

	diskSource := NewMockSource()
	diskSource.On("IsReady").Return(true).Once()

	defer diskSource.AssertExpectations(t)

	source := &autoSource{gitlab: gitlabSource, disk: diskSource}

	require.Equal(t, true, source.IsReady())
}

func TestDiskServerlessSourceRead(t *testing.T) {
	diskSource := NewMockSource()
	diskSource.On("Read", "gitlab.io").Once()

	defer diskSource.AssertExpectations(t)

	source := &diskServerlessSource{disk: diskSource}

	source.Read("gitlab.io")
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
