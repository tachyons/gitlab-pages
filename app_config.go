package main

type appConfig struct {
	Domain string

	RootCertificate []byte
	RootKey         []byte

	ListenHTTP     []uintptr
	ListenHTTPS    []uintptr
	ListenProxy    []uintptr
	MetricsAddress string

	HTTP2        bool
	RedirectHTTP bool
}
