package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
)

type keepAliveListener struct {
	net.Listener
	duration time.Duration
}

type keepAliveSetter interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

type listenerConfig struct {
	fd        uintptr
	isProxyV2 bool
	tlsConfig *tls.Config
	limiter   *netutil.Limiter
	handler   http.Handler
}

func (ln *keepAliveListener) Accept() (net.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	kc := conn.(keepAliveSetter)
	kc.SetKeepAlive(true)
	kc.SetKeepAlivePeriod(ln.duration)

	return conn, nil
}

func (a *theApp) listenAndServe(config listenerConfig) error {
	// create server
	server := &http.Server{Handler: config.handler, TLSConfig: config.tlsConfig}

	// ensure http2 is enabled even if TLSConfig is not null
	// See https://github.com/golang/go/blob/97cee43c93cfccded197cd281f0a5885cdb605b4/src/net/http/server.go#L2947-L2954
	if server.TLSConfig != nil {
		server.TLSConfig.NextProtos = append(server.TLSConfig.NextProtos, "h2")
	}

	server.ReadTimeout = a.config.Server.ReadTimeout
	server.ReadHeaderTimeout = a.config.Server.ReadHeaderTimeout
	server.WriteTimeout = a.config.Server.WriteTimeout

	l, err := net.FileListener(os.NewFile(config.fd, "[socket]"))

	if err != nil {
		return fmt.Errorf("failed to listen on FD %d: %v", config.fd, err)
	}

	if config.limiter != nil {
		l = netutil.SharedLimitListener(l, config.limiter)
	}

	l = &keepAliveListener{l, a.config.Server.ListenKeepAlive}

	if config.isProxyV2 {
		l = &proxyproto.Listener{
			Listener: l,
			Policy: func(upstream net.Addr) (proxyproto.Policy, error) {
				return proxyproto.REQUIRE, nil
			},
		}
	}

	if config.tlsConfig != nil {
		l = tls.NewListener(l, server.TLSConfig)
	}

	return server.Serve(l)
}
