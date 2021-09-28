package auth

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

// NewMiddleware returns middleware which handles authentication requests
func (a *Auth) AuthenticationMiddleware(handler http.Handler, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.TryAuthenticate(w, r, s) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// NewMiddleware returns middleware which handle authorization
func (a *Auth) AuthorizationMiddleware(handler http.Handler) http.Handler {
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
