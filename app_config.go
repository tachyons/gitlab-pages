package main

type appConfig struct {
	Domain                 string
	ArtifactsServer        string
	ArtifactsServerTimeout int
	RootCertificate        []byte
	RootKey                []byte
	AdminCertificate       []byte
	AdminKey               []byte
	AdminToken             []byte

	ListenHTTP       []uintptr
	ListenHTTPS      []uintptr
	ListenProxy      []uintptr
	ListenMetrics    uintptr
	ListenAdminUnix  uintptr
	ListenAdminHTTPS uintptr

	HTTP2        bool
	RedirectHTTP bool
	StatusPath   string

	DisableCrossOriginRequests bool

	LogFormat  string
	LogVerbose bool
}
