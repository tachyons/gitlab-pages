package errortracking

import (
	"net/http"

	raven "github.com/getsentry/raven-go"
)

// NewHandler will recover from panics inside handlers and reports the stacktrace to the errorreporting provider
func NewHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(raven.RecoveryHandler(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}))
}
