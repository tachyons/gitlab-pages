package request

import (
	"context"
	"net/http"
)

type ctxKey string

const (
	ctxHTTPSKey ctxKey = "https"
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
