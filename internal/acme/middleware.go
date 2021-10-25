package acme

import (
	"net/http"
	// TODO: break this dependency too
	d "gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// AcmeMiddleware handles ACME challenges
func (m *Middleware) AcmeMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := d.FromRequest(r)

		if m.ServeAcmeChallenges(w, r, domain) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}
