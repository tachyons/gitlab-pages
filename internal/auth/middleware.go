package auth

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

// NewMiddleware returns middleware which handles authentication requests
func NewMiddleware(handler http.Handler, a *Auth, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.TryAuthenticate(w, r, s) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}
