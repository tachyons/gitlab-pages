package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsHTTPS(t *testing.T) {
	tests := map[string]struct {
		u      string
		scheme string
	}{
		"when scheme is http": {
			u:      "/",
			scheme: SchemeHTTP,
		},
		"when scheme is https": {
			u:      "/",
			scheme: SchemeHTTPS,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.u, nil)
			req.URL.Scheme = test.scheme

			require.Equal(t, test.scheme == SchemeHTTPS, IsHTTPS(req))
		})
	}
}

func TestGetHostWithoutPort(t *testing.T) {
	tests := map[string]struct {
		u        string
		host     string
		expected string
	}{
		"when port component is provided": {
			u:        "https://example.com:443",
			host:     "my.example.com:8080",
			expected: "my.example.com",
		},
		"when port component is not provided": {
			u:        "http://example.com",
			host:     "my.example.com",
			expected: "my.example.com",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.u, nil)
			req.Host = test.host

			host := GetHostWithoutPort(req)
			require.Equal(t, test.expected, host)
		})
	}
}

func TestGetRemoteAddrWithoutPort(t *testing.T) {
	tests := map[string]struct {
		u          string
		remoteAddr string
		expected   string
	}{
		"when port component is provided": {
			u:          "https://example.com:443",
			remoteAddr: "127.0.0.1:1000",
			expected:   "127.0.0.1",
		},
		"when port component is not provided": {
			u:          "http://example.com",
			remoteAddr: "127.0.0.1",
			expected:   "127.0.0.1",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.u, nil)
			req.RemoteAddr = test.remoteAddr

			addr := GetRemoteAddrWithoutPort(req)
			require.Equal(t, test.expected, addr)
		})
	}
}
