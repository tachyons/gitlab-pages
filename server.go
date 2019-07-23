package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/context"
	"golang.org/x/net/http2"

	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/tlsconfig"
)

type keepAliveListener struct {
	net.Listener
}

type keepAliveSetter interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

func (ln *keepAliveListener) Accept() (net.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	kc := conn.(keepAliveSetter)
	kc.SetKeepAlive(true)
	kc.SetKeepAlivePeriod(3 * time.Minute)

	return conn, nil
}

func listenAndServe(fd uintptr, handler http.Handler, useHTTP2 bool, tlsConfig *tls.Config, limiter *netutil.Limiter) error {
	// create server
	server := &http.Server{Handler: context.ClearHandler(handler), TLSConfig: tlsConfig}

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

	if limiter != nil {
		l = netutil.SharedLimitListener(l, limiter)
	}

	if tlsConfig != nil {
		tlsListener := tls.NewListener(&keepAliveListener{l}, server.TLSConfig)
		return server.Serve(tlsListener)
	}
	return server.Serve(&keepAliveListener{l})
}

func listenAndServeTLS(fd uintptr, cert, key []byte, handler http.Handler, getCertificate tlsconfig.GetCertificateFunc, insecureCiphers bool, tlsMinVersion uint16, tlsMaxVersion uint16, useHTTP2 bool, limiter *netutil.Limiter) error {
	tlsConfig, err := tlsconfig.Create(cert, key, getCertificate, insecureCiphers, tlsMinVersion, tlsMaxVersion)
	if err != nil {
		return err
	}

	return listenAndServe(fd, handler, useHTTP2, tlsConfig, limiter)
}
