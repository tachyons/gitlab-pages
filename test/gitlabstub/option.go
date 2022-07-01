package gitlabstub

import (
	"crypto/tls"
	"net/http"
	"time"
)

type config struct {
	pagesHandler http.HandlerFunc
	pagesRoot    string
	delay        time.Duration
	tlsConfig    *tls.Config
}

type Option func(*config)

func defaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

func WithPagesHandler(ph http.HandlerFunc) Option {
	return func(sc *config) {
		sc.pagesHandler = ph
	}
}

func WithPagesRoot(pagesRoot string) Option {
	return func(sc *config) {
		sc.pagesRoot = pagesRoot
	}
}

func WithDelay(delay time.Duration) Option {
	return func(sc *config) {
		sc.delay = delay
	}
}

func WithCertificate(cert tls.Certificate) Option {
	return func(c *config) {
		if c.tlsConfig == nil {
			c.tlsConfig = defaultTLSConfig()
		}
		c.tlsConfig.Certificates = append(c.tlsConfig.Certificates, cert)
	}
}
