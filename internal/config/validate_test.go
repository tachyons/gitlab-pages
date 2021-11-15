package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         func(*Config)
		expectedErr error
	}{
		{
			name:        "no_listeners",
			cfg:         noListeners,
			expectedErr: ErrNoListener,
		},
		{
			name: "no_auth",
			cfg:  noAuth,
		},
		{
			name:        "auth_no_secret",
			cfg:         authNoSecret,
			expectedErr: ErrAuthNoSecret,
		},
		{
			name:        "auth_no_client_id",
			cfg:         authNoClientID,
			expectedErr: ErrAuthNoClientID,
		},
		{
			name:        "auth_no_client_secret",
			cfg:         authNoClientSecret,
			expectedErr: ErrAuthNoClientSecret,
		},
		{
			name:        "auth_no_gitlab_Server",
			cfg:         authNoPublicServer,
			expectedErr: ErrAuthNoGitlabServer,
		},
		{
			name:        "auth_no_redirect",
			cfg:         authNoRedirect,
			expectedErr: ErrAuthNoRedirect,
		},
		{
			name: "artifact_no_url",
			cfg:  artifactsNoURL,
		},
		{
			name:        "artifact_malformed_scheme",
			cfg:         artifactsMalformedScheme,
			expectedErr: ErrArtifactsServerUnsupportedScheme,
		},
		{
			name:        "artifact_invalid_timeout",
			cfg:         artifactsInvalidTimeout,
			expectedErr: ErrArtifactsServerInvalidTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.cfg(&cfg)

			err := Validate(&cfg)
			if tt.expectedErr != nil {
				require.True(t, errors.Is(err, tt.expectedErr))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func noListeners(cfg *Config) {
	cfg.ListenHTTPStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSStrings = MultiStringFlag{separator: ","}
	cfg.ListenProxyStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSProxyv2Strings = MultiStringFlag{separator: ","}
}

func noAuth(cfg *Config) {
	cfg.Authentication = Auth{}
}

func authNoSecret(cfg *Config) {
	cfg.Authentication.Secret = ""
}

func authNoClientID(cfg *Config) {
	cfg.Authentication.ClientID = ""
}

func authNoClientSecret(cfg *Config) {
	cfg.Authentication.ClientSecret = ""
}

func authNoPublicServer(cfg *Config) {
	cfg.GitLab.PublicServer = ""
}

func authNoRedirect(cfg *Config) {
	cfg.Authentication.RedirectURI = ""
}

func artifactsNoURL(cfg *Config) {
	cfg.ArtifactsServer.URL = ""
}

func artifactsMalformedScheme(cfg *Config) {
	cfg.ArtifactsServer.URL = "foo://example.com"
}

func artifactsInvalidTimeout(cfg *Config) {
	cfg.ArtifactsServer.TimeoutSeconds = -1
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
