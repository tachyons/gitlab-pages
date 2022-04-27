package main

import (
	"context"
	"crypto/tls"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

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

	server.ErrorLog = stdlog.New(logrus.StandardLogger().Writer(), "", 0)
	// ensure http2 is enabled even if TLSConfig is not null
	// See https://github.com/golang/go/blob/97cee43c93cfccded197cd281f0a5885cdb605b4/src/net/http/server.go#L2947-L2954
	if server.TLSConfig != nil {
		server.TLSConfig.NextProtos = append(server.TLSConfig.NextProtos, "h2")
	}

	server.ReadTimeout = a.config.Server.ReadTimeout
	server.ReadHeaderTimeout = a.config.Server.ReadHeaderTimeout
	server.WriteTimeout = a.config.Server.WriteTimeout

	lc := net.ListenConfig{
		KeepAlive: a.config.Server.ListenKeepAlive,
	}

	l, err := listenAddr(context.Background(), &lc, addr)
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

	log.WithFields(log.Fields{
		"config_addr": addr,
		"listen_addr": l.Addr(),
	}).Infof("server listening on: %s", l.Addr())

	return server.Serve(l)
}

func listenAddr(ctx context.Context, lc *net.ListenConfig, address string) (net.Listener, error) {
	if filepath.IsAbs(address) {
		return lc.Listen(ctx, "unix", address)
	}

	return lc.Listen(ctx, "tcp", address)
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
