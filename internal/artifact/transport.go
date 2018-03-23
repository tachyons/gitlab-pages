package artifact

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	sysPoolOnce = &sync.Once{}
	sysPool     *x509.CertPool

	transport = &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			return tls.Dial(network, addr, &tls.Config{RootCAs: pool()})
		},
	}
)

// This is here because macOS does not support the SSL_CERT_FILE
// environment variable. We have arrange things to read SSL_CERT_FILE as
// late as possible to avoid conflicts with file descriptor passing at
// startup.
func pool() *x509.CertPool {
	sysPoolOnce.Do(loadPool)
	return sysPool
}

func loadPool() {
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	if sslCertFile == "" {
		return
	}

	var err error
	sysPool, err = x509.SystemCertPool()
	if err != nil {
		log.WithError(err).Error("failed to load system cert pool for artifacts client")
		return
	}

	certPem, err := ioutil.ReadFile(sslCertFile)
	if err != nil {
		log.WithError(err).Error("failed to read SSL_CERT_FILE")
		return
	}

	sysPool.AppendCertsFromPEM(certPem)
}
