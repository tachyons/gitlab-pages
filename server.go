package main

import (
	"crypto/tls"
	"golang.org/x/net/http2"
	"net/http"
)

type TLSHandlerFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

func ListenAndServe(addr string, handler http.HandlerFunc) error {
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

func ListenAndServeTLS(addr string, certFile, keyFile string, handler http.HandlerFunc, tlsHandler TLSHandlerFunc) error {
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
