package acl

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

// NewMiddleware returns middleware which handle authorization
func NewMiddleware(handler http.Handler, a *auth.Auth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := request.GetDomain(r)

		// Only for projects that have access control enabled
		if domain.IsAccessControlEnabled(r) {
			// accessControlMiddleware
			if a.CheckAuthentication(w, r, domain) {
				return
			}
		}

		handler.ServeHTTP(w, r)
	})
}
