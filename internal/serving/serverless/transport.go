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
		// TODO
		// if address == domain+":443" {
		// 	address = cluster + ":443"
		// }

		return dialer.DialContext(ctx, network, address)
	}

	return &Transport{
		transport: &http.Transport{
			DialContext:         dialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			// TODO TLSClientConfig:     newTLSConfig(),
		},
	}
}

// RoundTrip performs a connection to a Knative cluster and returns a response
func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.transport.RoundTrip(request)

	return response, err
}
