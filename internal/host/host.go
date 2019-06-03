package host

import (
	"net"
	"net/http"
	"strings"
)

// FromString tries to split host port from string, returns host or initial string if fail
func FromString(s string) string {
	host := strings.ToLower(s)

	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}

	return host
}

// FromRequest tries to split host port from r.Host, returns host or initial string if fail
func FromRequest(r *http.Request) string {
	return FromString(r.Host)
}
