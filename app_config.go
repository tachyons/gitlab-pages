package main

type appConfig struct {
	Domain  string
	RootDir string

	RootCertificate []byte
	RootKey         []byte

	ListenHTTP  uintptr
	ListenHTTPS uintptr
	listenProxy uintptr

	HTTP2     bool
	ServeHTTP bool
}
