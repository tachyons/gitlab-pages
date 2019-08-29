package request

import (
	"context"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type ctxKey string

const (
	ctxDomainKey ctxKey = "domain"
)

// WithDomain saves the domain in the request's context
func WithDomain(r *http.Request, domain *domain.D) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxDomainKey, domain)

	return r.WithContext(ctx)
}

// GetDomain extracts the domain from request's context
func GetDomain(r *http.Request) *domain.D {
	return r.Context().Value(ctxDomainKey).(*domain.D)
}
