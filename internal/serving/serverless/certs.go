package serverless

import (
	"crypto/tls"
	"crypto/x509"
)

// Certs holds definition of certificates we use to perform mTLS
// handshake with a cluster
type Certs struct {
	RootCerts   *x509.CertPool
	Certificate tls.Certificate
}

// NewClusterCerts creates a new cluster configuration from cert / key pair
func NewClusterCerts(clientCert, clientKey string) (*Certs, error) {
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(clientCert))

	return &Certs{RootCerts: caCertPool, Certificate: cert}, nil
}
