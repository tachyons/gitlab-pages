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
			cfg:         NoListeners,
			expectedErr: ErrNoListener,
		},
		{
			name: "no_auth",
			cfg:  NoAuth,
		},
		{
			name:        "auth_no_secret",
			cfg:         AuthNoSecret,
			expectedErr: ErrAuthNoSecret,
		},
		{
			name:        "auth_no_client_id",
			cfg:         AuthNoClientID,
			expectedErr: ErrAuthNoClientID,
		},
		{
			name:        "auth_no_client_secret",
			cfg:         AuthNoClientSecret,
			expectedErr: ErrAuthNoClientSecret,
		},
		{
			name:        "auth_no_gitlab_Server",
			cfg:         AuthNoPublicServer,
			expectedErr: ErrAuthNoGitlabServer,
		},
		{
			name:        "auth_no_redirect",
			cfg:         AuthNoRedirect,
			expectedErr: ErrAuthNoRedirect,
		},
		{
			name: "artifact_no_url",
			cfg:  ArtifactsNoURL,
		},
		{
			name:        "artifact_malformed_scheme",
			cfg:         ArtifactsMalformedScheme,
			expectedErr: ErrArtifactsServerUnsupportedScheme,
		},
		{
			name:        "artifact_invalid_timeout",
			cfg:         ArtifactsInvalidTimeout,
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

func NoListeners(cfg *Config) {
	cfg.ListenHTTPStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSStrings = MultiStringFlag{separator: ","}
	cfg.ListenProxyStrings = MultiStringFlag{separator: ","}
	cfg.ListenHTTPSProxyv2Strings = MultiStringFlag{separator: ","}
}

func NoAuth(cfg *Config) {
	cfg.Authentication = Auth{}
}

func AuthNoSecret(cfg *Config) {
	cfg.Authentication.Secret = ""
}

func AuthNoClientID(cfg *Config) {
	cfg.Authentication.ClientID = ""
}

func AuthNoClientSecret(cfg *Config) {
	cfg.Authentication.ClientSecret = ""
}

func AuthNoPublicServer(cfg *Config) {
	cfg.GitLab.PublicServer = ""
}

func AuthNoRedirect(cfg *Config) {
	cfg.Authentication.RedirectURI = ""
}

func ArtifactsNoURL(cfg *Config) {
	cfg.ArtifactsServer.URL = ""
}

func ArtifactsMalformedScheme(cfg *Config) {
	cfg.ArtifactsServer.URL = "foo://example.com"
}

func ArtifactsInvalidTimeout(cfg *Config) {
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
