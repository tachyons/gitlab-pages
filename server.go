package main

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"os"
	"time"
)

type tlsHandlerFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func listenAndServe(fd uintptr, handler http.HandlerFunc, useHTTP2 bool, tlsConfig *tls.Config) error {
	// create server
	server := &http.Server{Handler: handler, TLSConfig: tlsConfig}

	if useHTTP2 {
		err := http2.ConfigureServer(server, &http2.Server{})
		if err != nil {
			return err
		}
	}

	l, err := net.FileListener(os.NewFile(fd, "[socket]"))
	if err != nil {
		return fmt.Errorf("failed to listen on FD %d: %v", fd, err)
	}

	if tlsConfig != nil {
		tlsListener := tls.NewListener(tcpKeepAliveListener{l.(*net.TCPListener)}, server.TLSConfig)
		return server.Serve(tlsListener)
	}
	return server.Serve(&tcpKeepAliveListener{l.(*net.TCPListener)})
}

func listenAndServeTLS(fd uintptr, cert, key []byte, handler http.HandlerFunc, tlsHandler tlsHandlerFunc, useHTTP2 bool) error {
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}
	tlsConfig.GetCertificate = tlsHandler
	tlsConfig.NextProtos = []string{
		"http/1.1",
	}
	tlsConfig.Certificates = []tls.Certificate{
		certificate,
	}
	return listenAndServe(fd, handler, useHTTP2, tlsConfig)
}
