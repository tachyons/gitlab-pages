package serverless

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Transport is a struct that handle the proxy connection round trip to Knative
// cluster
type Transport struct {
	cluster   Cluster
	transport *http.Transport
}

// NewTransport fabricates as new transport type
func NewTransport(cluster Cluster) *Transport {
	dialer := net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}

	dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		address = cluster.Host()

		return dialer.DialContext(ctx, network, address)
	}

	return &Transport{
		cluster: cluster,
		transport: &http.Transport{
			DialContext:         dialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig:     cluster.TLSConfig(),
		},
	}
}

// RoundTrip performs a connection to a Knative cluster and returns a response
func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.transport.RoundTrip(request)

	// TODO add prometheus metrics for round trip timing
	return response, err
}
