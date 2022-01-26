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
}

type keepAliveSetter interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

type listenerConfig struct {
	isProxyV2 bool
	tlsConfig *tls.Config
	limiter   *netutil.Limiter
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

func (a *theApp) listenAndServe(server *http.Server, fd uintptr, h http.Handler, opts ...option) error {
	config := &listenerConfig{}

	for _, opt := range opts {
		opt(config)
	}

	// create server
	server.Handler = h
	server.TLSConfig = config.tlsConfig

	// ensure http2 is enabled even if TLSConfig is not null
	// See https://github.com/golang/go/blob/97cee43c93cfccded197cd281f0a5885cdb605b4/src/net/http/server.go#L2947-L2954
	if server.TLSConfig != nil {
		server.TLSConfig.NextProtos = append(server.TLSConfig.NextProtos, "h2")
	}

	l, err := net.FileListener(os.NewFile(fd, "[socket]"))
	if err != nil {
		return fmt.Errorf("failed to listen on FD %d: %w", fd, err)
	}

	if config.limiter != nil {
		l = netutil.SharedLimitListener(l, config.limiter)
	}

	l = &keepAliveListener{l}

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

type option func(*listenerConfig)

func withProxyV2() option {
	return func(conf *listenerConfig) {
		conf.isProxyV2 = true
	}
}

func withTLSConfig(c *tls.Config) option {
	return func(conf *listenerConfig) {
		conf.tlsConfig = c
	}
}

func withLimiter(l *netutil.Limiter) option {
	return func(conf *listenerConfig) {
		conf.limiter = l
	}
}
