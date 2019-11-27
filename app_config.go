package main

type appConfig struct {
	Domain                 string
	ArtifactsServer        string
	ArtifactsServerTimeout int
	RootCertificate        []byte
	RootKey                []byte
	MaxConns               int

	ListenHTTP      []uintptr
	ListenHTTPS     []uintptr
	ListenProxy     []uintptr
	ListenMetrics   uintptr
	InsecureCiphers bool
	TLSMinVersion   uint16
	TLSMaxVersion   uint16

	HTTP2        bool
	RedirectHTTP bool
	StatusPath   string

	DisableCrossOriginRequests bool

	LogFormat  string
	LogVerbose bool

	StoreSecret        string
	GitLabServer       string
	GitLabAPISecretKey []byte
	ClientID           string
	ClientSecret       string
	RedirectURI        string
	SentryDSN          string
	SentryEnvironment  string
	CustomHeaders      []string
}
