package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	sysPoolOnce = &sync.Once{}
	sysPool     *x509.CertPool

	// InternalTransport can be used with http.Client with TLS and certificates
	InternalTransport = newInternalTransport()
)

type meteredRoundTripper struct {
	next      http.RoundTripper
	durations *prometheus.GaugeVec
	counter   *prometheus.CounterVec
}

func newInternalTransport() *http.Transport {
	return &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			return tls.Dial(network, addr, &tls.Config{RootCAs: pool()})
		},
		Proxy: http.ProxyFromEnvironment,
		// overrides the DefaultMaxIdleConnsPerHost = 2
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
}

// NewTransportWithMetrics will create a custom http.RoundTripper that can be used with an http.Client.
// The RoundTripper will report metrics based on the collectors passed.
func NewTransportWithMetrics(gaugeVec *prometheus.GaugeVec, counterVec *prometheus.CounterVec) http.RoundTripper {
	return &meteredRoundTripper{
		next:      InternalTransport,
		durations: gaugeVec,
		counter:   counterVec,
	}
}

// This is here because macOS does not support the SSL_CERT_FILE
// environment variable. We have arrange things to read SSL_CERT_FILE as
// late as possible to avoid conflicts with file descriptor passing at
// startup.
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

	// Try to load from SSL_CERT_FILE
	// TODO: Handle SSL_CERT_DIR?
	// See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/415
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	if sslCertFile != "" {
		certPem, err := ioutil.ReadFile(sslCertFile)
		if err != nil {
			log.WithError(err).Error("failed to read SSL_CERT_FILE")
			return
		}
		sysPool.AppendCertsFromPEM(certPem)
	}

	if err := loadCertDir(); err != nil {
		log.WithError(err).Warn("failed to load SSL_CERT_DIR")
	}

}

func loadCertDir() error {
	sslCertDir := os.Getenv("SSL_CERT_DIR")
	if sslCertDir == "" {
		return nil
	}

	entries, err := ioutil.ReadDir(sslCertDir)
	if err != nil {
		return fmt.Errorf("failed to read SSL_CERT_DIR: %w", err)
	}

	for _, fi := range entries {
		// Copy only regular files and symlinks
		mode := fi.Mode()
		if !(mode.IsRegular() || mode&os.ModeSymlink != 0) {
			continue
		}
		fmt.Printf("the fi: %q thee dir: %q\n\n\n\n", fi.Name(), sslCertDir)
		cert, err := ioutil.ReadFile(sslCertDir + "/" + fi.Name())
		if err != nil {
			log.WithError(err).Warnf("failed to open cert, skipping: %q", fi.Name())
			panic(2)
			continue
		}

		ok := sysPool.AppendCertsFromPEM(cert)
		if !ok {
			panic("didnet???/")
			log.Warnf("failed to append to sysPool, skipping: %q", fi.Name())
		}
	}

	return nil
}

// withRoundTripper takes an original RoundTripper, reports metrics based on the
// gauge and counter collectors passed
func (mrt *meteredRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := mrt.next.RoundTrip(r)
	if err != nil {
		mrt.counter.WithLabelValues("error").Inc()
		return nil, err
	}

	statusCode := strconv.Itoa(resp.StatusCode)
	mrt.durations.WithLabelValues(statusCode).Set(time.Since(start).Seconds())
	mrt.counter.WithLabelValues(statusCode).Inc()

	return resp, nil
}
