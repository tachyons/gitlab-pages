package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidConfig(t *testing.T) {
	cfg := validConfig()
	err := Validate(&cfg)

	require.NoError(t, err)
}

func TestNoListeners(t *testing.T) {
	cfg := validConfig()
	cfg.ListenHTTPStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSStrings = MultiStringFlag{separator: ","}
	cfg.ListenProxyStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSProxyv2Strings = MultiStringFlag{separator: ","}
	err := Validate(&cfg)

	require.Equal(t, err, ErrNoListener)
}

func TestNoAuth(t *testing.T) {
	cfg := validConfig()
	cfg.Authentication = Auth{}
	err := Validate(&cfg)

	require.NoError(t, err)
}

func TestAuthNoSecret(t *testing.T) {
	cfg := validConfig()
	cfg.Authentication.Secret = ""
	err := Validate(&cfg)

	require.True(t, errors.Is(err, ErrAuthNoSecret))
}

func TestAuthNoClientID(t *testing.T) {
	cfg := validConfig()
	cfg.Authentication.ClientID = ""
	err := Validate(&cfg)

	require.True(t, errors.Is(err, ErrAuthNoClientID))
}

func TestAuthNoClientSecret(t *testing.T) {
	cfg := validConfig()
	cfg.Authentication.ClientSecret = ""
	err := Validate(&cfg)

	require.True(t, errors.Is(err, ErrAuthNoClientSecret))
}

func TestAuthNoPublicServer(t *testing.T) {
	cfg := validConfig()
	cfg.GitLab.PublicServer = ""
	err := Validate(&cfg)

	require.True(t, errors.Is(err, ErrAuthNoGitlabServer))
}

func TestAuthNoRedirect(t *testing.T) {
	cfg := validConfig()
	cfg.Authentication.RedirectURI = ""
	err := Validate(&cfg)

	require.True(t, errors.Is(err, ErrAuthNoRedirect))
}

func TestArtifactsNoURL(t *testing.T) {
	cfg := validConfig()
	cfg.ArtifactsServer.URL = ""
	err := Validate(&cfg)

	require.NoError(t, err)
}

func TestArtifactsMalformedURL(t *testing.T) {
	cfg := validConfig()
	cfg.ArtifactsServer.URL = ":foo"
	err := Validate(&cfg)

	require.Error(t, err)
}

func TestArtifactsMalformedScheme(t *testing.T) {
	cfg := validConfig()
	cfg.ArtifactsServer.URL = "foo://example.com"
	err := Validate(&cfg)

	require.Equal(t, err, ErrArtifactsServerUnsupportedScheme)
}

func TestArtifactsInvalidTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.ArtifactsServer.TimeoutSeconds = -1
	err := Validate(&cfg)

	require.Equal(t, err, ErrArtifactsServerInvalidTimeout)
}

func validConfig() Config {
	cfg := Config{
		ListenHTTPStrings: MultiStringFlag{
			value:     []string{"127.0.0.1:80"},
			separator: ",",
		},
		ArtifactsServer: ArtifactsServer{
			URL:            "https://example.com",
			TimeoutSeconds: 1,
		},
		Authentication: Auth{
			Secret:       "foo",
			ClientID:     "bar",
			ClientSecret: "bar-secret",
			RedirectURI:  "https://example.com",
		},
		GitLab: GitLab{
			PublicServer: "https://gitlab.example.com",
		},
	}

	return cfg
}
