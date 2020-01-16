package serverless

import (
	"crypto/tls"
	"crypto/x509"
)

// Cluster represent a Knative cluster that we want to proxy requests to
type Cluster struct {
	Address  string
	Hostname string
	Port     string
	Config   *Config
}

// Config holds configuration for a cluster, especially definition of
// certificates we use to perform mTLS handshake
type Config struct {
	RootCerts   *x509.CertPool
	Certificate tls.Certificate
}

// NewClusterConfig creates a new cluster configuration from cert / key pair
func NewClusterConfig(clientCert, clientKey string) (*Config, error) {
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(clientCert))

	return &Config{RootCerts: caCertPool, Certificate: cert}, nil
}

// TLSConfig builds a new tls.Config and returns a pointer to it
func (c Cluster) TLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{c.Config.Certificate},
		RootCAs:      c.Config.RootCerts,
		ServerName:   c.Hostname,
	}
}
