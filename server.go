package main

import (
	"crypto/tls"
	"golang.org/x/net/http2"
	"net/http"
)

type tlsHandlerFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

func listenAndServe(addr string, handler http.HandlerFunc) error {
	// create server
	server := &http.Server{Addr: addr, Handler: handler}

	if *http2proto {
		err := http2.ConfigureServer(server, &http2.Server{})
		if err != nil {
			return err
		}
	}

	return server.ListenAndServe()
}

func listenAndServeTLS(addr string, certFile, keyFile string, handler http.HandlerFunc, tlsHandler tlsHandlerFunc) error {
	// create server
	server := &http.Server{Addr: addr, Handler: handler}
	server.TLSConfig = &tls.Config{}
	server.TLSConfig.GetCertificate = tlsHandler

	if *http2proto {
		err := http2.ConfigureServer(server, &http2.Server{})
		if err != nil {
			return err
		}
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}
