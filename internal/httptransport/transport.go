package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// DefaultTTFBTimeout is the timeout used in the meteredRoundTripper
	// when calling http.Transport.RoundTrip. The request will be cancelled
	// if the response takes longer than this.
	DefaultTTFBTimeout = 15 * time.Second
)

var (
	sysPoolOnce = &sync.Once{}
	sysPool     *x509.CertPool

	// only overridden by transport_darwin.go
	loadExtraCerts = func() {}
	// DefaultTransport can be used with http.Client with TLS and certificates
	DefaultTransport = NewTransport()
)

// NewTransport initializes an http.Transport with a custom dialer that includes TLS Root CAs.
// It sets default connection values such as timeouts and max idle connections.
func NewTransport() *http.Transport {
	return &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			return tls.Dial(network, addr, &tls.Config{RootCAs: pool()})
		},
		Proxy: http.ProxyFromEnvironment,
		// overrides the DefaultMaxIdleConnsPerHost = 2
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		// Set more timeouts https://gitlab.com/gitlab-org/gitlab-pages/-/issues/495
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 15 * time.Second,
	}
}

// This is here because macOS does not support the SSL_CERT_FILE and
// SSL_CERT_DIR environment variables. We have arranged things to read
// SSL_CERT_FILE and SSL_CERT_DIR  as late as possible to avoid conflicts
// with file descriptor passing at startup.
func pool() *x509.CertPool {
	sysPoolOnce.Do(loadPool)
	return sysPool
}

func loadPool() {
	var err error

	// Always load the system cert pool
	sysPool, err = x509.SystemCertPool()
	if err != nil {
		log.WithError(err).Error("failed to load system cert pool for http client")
		return
	}

	// Go does not load SSL_CERT_FILE and SSL_CERT_DIR on darwin systems so we need to
	// load them manually in OSX. See https://golang.org/src/crypto/x509/root_unix.go
	loadExtraCerts()
}
