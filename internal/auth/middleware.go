package auth

import (
	"net/http"

	d "gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

// AuthenticationMiddleware handles authentication requests
func (a *Auth) AuthenticationMiddleware(handler http.Handler, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.TryAuthenticate(w, r, s) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// AuthorizationMiddleware handles authorization
func (a *Auth) AuthorizationMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := d.FromRequest(r)

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
