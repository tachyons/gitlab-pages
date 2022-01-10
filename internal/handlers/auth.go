package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func (h *Handlers) Authorization(handler http.Handler) http.Handler {
	return h.Auth.AuthorizationMiddleware(handler)
}

func Authentication(auth *auth.Auth, s source.Source, handler http.Handler) http.Handler {
	return auth.AuthenticationMiddleware(handler, s)
}

// CheckAuthAndServeNotFound performs the auth process if domain can't be found
// the main purpose of this process is to avoid leaking the project existence/not-existence
// by behaving the same if user has no access to the project or if project simply does not exists
func CheckAuthAndServeNotFound(a *auth.Auth, domain *domain.Domain, w http.ResponseWriter, r *http.Request) {
	// To avoid user knowing if pages exist, we will force user to login and authorize pages
	if a.CheckAuthenticationWithoutProject(w, r, domain) {
		return
	}

	// auth succeeded try to serve the correct 404 page
	domain.ServeNotFoundAuthFailed(w, r)
}
