package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/context"
	proxyproto "github.com/pires/go-proxyproto"
	"golang.org/x/net/http2"

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
	isTLS     bool
	isProxyV2 bool
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

func (a *theApp) listenAndServe(fd uintptr, handler http.Handler, limiter *netutil.Limiter, config listenerConfig) error {
	var tlsConfig *tls.Config
	if config.isTLS {
		var err error
		if tlsConfig, err = a.TLSConfig(); err != nil {
			return err
		}
	}
	// create server
	server := &http.Server{Handler: context.ClearHandler(handler), TLSConfig: tlsConfig}

	if a.config.General.HTTP2 {
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

	l = &keepAliveListener{l}

	if config.isProxyV2 {
		l = &proxyproto.Listener{
			Listener: l,
			Policy: func(upstream net.Addr) (proxyproto.Policy, error) {
				return proxyproto.REQUIRE, nil
			},
		}
	}

	if tlsConfig != nil {
		l = tls.NewListener(l, server.TLSConfig)
	}

	return server.Serve(l)
}
