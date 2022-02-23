package config

import (
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
			expectedErr: errNoListener,
		},
		{
			name: "no_auth",
			cfg:  noAuth,
		},
		{
			name:        "auth_no_secret",
			cfg:         authNoSecret,
			expectedErr: errAuthNoSecret,
		},
		{
			name:        "auth_no_client_id",
			cfg:         authNoClientID,
			expectedErr: errAuthNoClientID,
		},
		{
			name:        "auth_no_client_secret",
			cfg:         authNoClientSecret,
			expectedErr: errAuthNoClientSecret,
		},
		{
			name:        "auth_no_gitlab_Server",
			cfg:         authNoPublicServer,
			expectedErr: errAuthNoGitlabServer,
		},
		{
			name:        "auth_no_redirect",
			cfg:         authNoRedirect,
			expectedErr: errAuthNoRedirect,
		},
		{
			name: "artifact_no_url",
			cfg:  artifactsNoURL,
		},
		{
			name:        "artifact_malformed_scheme",
			cfg:         artifactsMalformedScheme,
			expectedErr: errArtifactsServerUnsupportedScheme,
		},
		{
			name:        "artifact_invalid_timeout",
			cfg:         artifactsInvalidTimeout,
			expectedErr: errArtifactsServerInvalidTimeout,
		},
		{
			name:        "empty_listener",
			cfg:         emptyListeners,
			expectedErr: errEmptyListener,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.cfg(&cfg)

			err := Validate(&cfg)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func emptyListeners(cfg *Config) {
	cfg.ListenHTTPSStrings = MultiStringFlag{
		value:     []string{"127.0.0.1:8080", "", ":8081"},
		separator: ",",
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
			value:     []string{"127.0.0.1:80", ":8081"},
			separator: ",",
		},
		ListenHTTPSStrings: MultiStringFlag{
			value:     []string{"127.0.0.1:", ":8082"},
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

func TestValidTLSVersions(t *testing.T) {
	tests := map[string]struct {
		tlsMin string
		tlsMax string
	}{
		"tls 1.3 only": {tlsMin: "tls1.3", tlsMax: "tls1.3"},
		"tls 1.2 only": {tlsMin: "tls1.2", tlsMax: "tls1.2"},
		"tls 1.3 max":  {tlsMax: "tls1.3"},
		"tls 1.2 max":  {tlsMax: "tls1.2"},
		"tls 1.3+":     {tlsMin: "tls1.3"},
		"tls 1.2+":     {tlsMin: "tls1.2"},
		"default":      {},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateTLSVersions(tc.tlsMin, tc.tlsMax)
			require.NoError(t, err)
		})
	}
}

func TestInvalidTLSVersions(t *testing.T) {
	tests := map[string]struct {
		tlsMin string
		tlsMax string
		err    string
	}{
		"invalid minimum TLS version": {tlsMin: "tls123", tlsMax: "", err: "invalid minimum TLS version: tls123"},
		"invalid maximum TLS version": {tlsMin: "", tlsMax: "tls123", err: "invalid maximum TLS version: tls123"},
		"TLS versions conflict":       {tlsMin: "tls1.3", tlsMax: "tls1.2", err: "invalid maximum TLS version: tls1.2; should be at least tls1.3"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateTLSVersions(tc.tlsMin, tc.tlsMax)
			require.EqualError(t, err, tc.err)
		})
	}
}
