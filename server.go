package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
)

type listenerConfig struct {
	isProxyV2 bool
	tlsConfig *tls.Config
	limiter   *netutil.Limiter
}

func (a *theApp) listenAndServe(server *http.Server, addr string, h http.Handler, opts ...option) error {
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

	lc := net.ListenConfig{
		KeepAlive: 3 * time.Minute,
	}

	l, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on addr %s: %w", addr, err)
	}

	if config.limiter != nil {
		l = netutil.SharedLimitListener(l, config.limiter)
	}

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
