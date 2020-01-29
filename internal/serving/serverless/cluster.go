package serverless

import (
	"crypto/tls"
	"strings"
)

// Cluster represent a Knative cluster that we want to proxy requests to
type Cluster struct {
	Address string // Address is a real IP address of a cluster ingress
	Port    string // Port is a real port of HTTP TLS service
	Name    string // Name is a cluster name, used in cluster certificates
	Certs   *Certs
}

// Host returns a real cluster location based on IP address and port
func (c Cluster) Host() string {
	return strings.Join([]string{c.Address, c.Port}, ":")
}

// TLSConfig builds a new tls.Config and returns a pointer to it
func (c Cluster) TLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{c.Certs.Certificate},
		RootCAs:      c.Certs.RootCerts,
		ServerName:   c.Name,
	}
}
