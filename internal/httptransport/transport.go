package httptransport

import (
	"crypto/tls"
	"crypto/x509"
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
		ForceAttemptHTTP2:   true,
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
