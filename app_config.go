package main

import "time"

type appConfig struct {
	PagesRoot              string
	Domain                 string
	ArtifactsServer        string
	ArtifactsServerTimeout int
	RootCertificate        []byte
	RootKey                []byte
	MaxConns               int

	ListenHTTP         []uintptr
	ListenHTTPS        []uintptr
	ListenProxy        []uintptr
	ListenHTTPSProxyv2 []uintptr
	ListenMetrics      uintptr
	InsecureCiphers    bool
	TLSMinVersion      uint16
	TLSMaxVersion      uint16

	HTTP2        bool
	RedirectHTTP bool
	StatusPath   string

	DisableCrossOriginRequests bool

	LogFormat  string
	LogVerbose bool

	StoreSecret               string
	GitLabServer              string
	InternalGitLabServer      string
	GitLabAPISecretKey        []byte
	GitlabClientHTTPTimeout   time.Duration
	GitlabJWTTokenExpiration  time.Duration
	DomainConfigurationSource string
	ClientID                  string
	ClientSecret              string
	RedirectURI               string
	SentryDSN                 string
	SentryEnvironment         string
	CustomHeaders             []string

	ZipCacheExpiry  time.Duration
	ZipCacheRefresh time.Duration
	ZipCacheCleanup time.Duration
	ZipeOpenTimeout time.Duration
}

// InternalGitLabServerURL returns URL to a GitLab instance.
func (config appConfig) InternalGitLabServerURL() string {
	return config.InternalGitLabServer
}

// GitlabClientSecret returns GitLab server access token.
func (config appConfig) GitlabAPISecret() []byte {
	return config.GitLabAPISecretKey
}

func (config appConfig) GitlabClientConnectionTimeout() time.Duration {
	return config.GitlabClientHTTPTimeout
}

func (config appConfig) GitlabJWTTokenExpiry() time.Duration {
	return config.GitlabJWTTokenExpiration
}

func (config appConfig) DomainConfigSource() string {
	return config.DomainConfigurationSource
}
