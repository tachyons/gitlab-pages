package main

type appConfig struct {
	Domain                 string
	ArtifactsServer        string
	ArtifactsServerTimeout int
	RootCertificate        []byte
	RootKey                []byte

	ListenHTTP    []uintptr
	ListenHTTPS   []uintptr
	ListenProxy   []uintptr
	ListenMetrics uintptr

	HTTP2        bool
	RedirectHTTP bool
	StatusPath   string

	DisableCrossOriginRequests bool
}
