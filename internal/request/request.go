package request

import (
	"context"
	"net"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type ctxKey string

const (
	ctxHostKey   ctxKey = "host"
	ctxDomainKey ctxKey = "domain"

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

// WithHostAndDomain saves host name and domain in the request's context
func WithHostAndDomain(r *http.Request, host string, domain *domain.Domain) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxHostKey, host)
	ctx = context.WithValue(ctx, ctxDomainKey, domain)

	return r.WithContext(ctx)
}

// GetHost extracts the host from request's context
func GetHost(r *http.Request) string {
	return r.Context().Value(ctxHostKey).(string)
}

// GetDomain extracts the domain from request's context
func GetDomain(r *http.Request) *domain.Domain {
	return r.Context().Value(ctxDomainKey).(*domain.Domain)
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
