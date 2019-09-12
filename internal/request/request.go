package request

import (
	"context"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type ctxKey string

const (
	ctxHTTPSKey  ctxKey = "https"
	ctxHostKey   ctxKey = "host"
	ctxDomainKey ctxKey = "domain"
)

// WithHTTPSFlag saves https flag in request's context
func WithHTTPSFlag(r *http.Request, https bool) *http.Request {
	ctx := context.WithValue(r.Context(), ctxHTTPSKey, https)

	return r.WithContext(ctx)
}

// IsHTTPS restores https flag from request's context
func IsHTTPS(r *http.Request) bool {
	return r.Context().Value(ctxHTTPSKey).(bool)
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
