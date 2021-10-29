package acme

import (
	"net/http"
	// TODO: break this dependency too https://gitlab.com/gitlab-org/gitlab-pages/-/issues/650
	domainCfg "gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// AcmeMiddleware handles ACME challenges
func (m *Middleware) AcmeMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := domainCfg.FromRequest(r)

		if m.ServeAcmeChallenges(w, r, domain) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}
