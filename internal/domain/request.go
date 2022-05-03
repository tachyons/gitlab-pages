package domain

import (
	"context"
	"net/http"
)

type ctxKey string

const (
	ctxDomainKey ctxKey = "domain"
)

// ReqWithDomain saves domain in the request's context
func ReqWithDomain(r *http.Request, domain *Domain) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxDomainKey, domain)

	return r.WithContext(ctx)
}

// FromRequest extracts the domain from request's context
func FromRequest(r *http.Request) *Domain {
	return r.Context().Value(ctxDomainKey).(*Domain)
}
