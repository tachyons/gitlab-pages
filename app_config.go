package main

type appConfig struct {
	Domain string

	RootCertificate []byte
	RootKey         []byte

	ListenHTTP  uintptr
	ListenHTTPS uintptr
	listenProxy uintptr

	HTTP2        bool
	RedirectHTTP bool
}
