package serverless

import (
	"crypto/tls"
	"crypto/x509"
)

// Cluster represent a Knative cluster that we want to proxy requests to
type Cluster struct {
	Address  string
	Port     string
	Hostname string
	Certs    *ClusterCerts
}

// ClusterCerts holds definition of certificates we use to perform mTLS
// handshake
type ClusterCerts struct {
	RootCerts   *x509.CertPool
	Certificate tls.Certificate
}

// NewClusterCerts creates a new cluster configuration from cert / key pair
func NewClusterCerts(clientCert, clientKey string) (*ClusterCerts, error) {
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(clientCert))

	return &ClusterCerts{RootCerts: caCertPool, Certificate: cert}, nil
}

// TLSConfig builds a new tls.Config and returns a pointer to it
func (c Cluster) TLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{c.Certs.Certificate},
		RootCAs:      c.Certs.RootCerts,
		ServerName:   c.Hostname,
	}
}
