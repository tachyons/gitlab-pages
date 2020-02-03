package request

import (
	"context"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type ctxKey string

const (
	ctxHTTPSKey  ctxKey = "https"
	ctxHostKey   ctxKey = "host"
	ctxDomainKey ctxKey = "domain"

	// SchemeHTTP name for the HTTP scheme
	SchemeHTTP = "http"
	// SchemeHTTPS name for the HTTPS scheme
	SchemeHTTPS = "https"
)

// WithHTTPSFlag saves https flag in request's context
func WithHTTPSFlag(r *http.Request, https bool) *http.Request {
	ctx := context.WithValue(r.Context(), ctxHTTPSKey, https)

	return r.WithContext(ctx)
}

// IsHTTPS checks whether the request originated from HTTP or HTTPS.
// It reads the ctxHTTPSKey from the context and returns its value
// It also checks that r.URL.Scheme matches the value in ctxHTTPSKey for HTTPS requests
// TODO: remove the ctxHTTPSKey from the context https://gitlab.com/gitlab-org/gitlab-pages/issues/219
func IsHTTPS(r *http.Request) bool {
	https := r.Context().Value(ctxHTTPSKey).(bool)

	if https != (r.URL.Scheme == SchemeHTTPS) {
		log.WithFields(log.Fields{
			"ctxHTTPSKey": https,
			"scheme":      r.URL.Scheme,
		}).Warn("request: r.URL.Scheme does not match value in ctxHTTPSKey")
	}

	// Returning the value of ctxHTTPSKey for now, can just switch to r.URL.Scheme == SchemeHTTPS later
	// and later can remove IsHTTPS method altogether
	return https
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
