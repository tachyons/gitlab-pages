package request

import (
	"net"
	"net/http"
)

const (
	// SchemeHTTP name for the HTTP scheme
	SchemeHTTP = "http"
	// SchemeHTTPS name for the HTTPS scheme
	SchemeHTTPS = "https"
)

// IsHTTPS checks whether the request originated from HTTP or HTTPS.
// It checks the value from r.URL.Scheme
func IsHTTPS(r *http.Request) bool {
	return r.URL.Scheme == SchemeHTTPS
}

// GetHostWithoutPort returns a host without the port. The host(:port) comes
// from a Host: header if it is provided, otherwise it is a server name.
func GetHostWithoutPort(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		return r.Host
	}

	return host
}

// GetRemoteAddrWithoutPort strips the port from the r.RemoteAddr
func GetRemoteAddrWithoutPort(r *http.Request) string {
	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return remoteAddr
}
