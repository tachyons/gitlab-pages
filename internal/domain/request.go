package domain

import (
	"context"
	"net/http"
)

type ctxKey string

const (
	ctxHostKey   ctxKey = "host"
	ctxDomainKey ctxKey = "domain"
)

// ReqWithHostAndDomain saves host name and domain in the request's context
func ReqWithHostAndDomain(r *http.Request, host string, domain *Domain) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxHostKey, host)
	ctx = context.WithValue(ctx, ctxDomainKey, domain)

	return r.WithContext(ctx)
}

// GetHost extracts the host from request's context
func GetHost(r *http.Request) string {
	return r.Context().Value(ctxHostKey).(string)
}

// FromRequest extracts the domain from request's context
func FromRequest(r *http.Request) *Domain {
	return r.Context().Value(ctxDomainKey).(*Domain)
}
