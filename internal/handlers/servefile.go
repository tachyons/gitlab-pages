package handlers

import (
	"net/http"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// ServeFileOrNotFoundHandler will serve static content or
// return a 404 Not Found response
func ServeFileOrNotFoundHandler(auth *auth.Auth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer metrics.ServingTime.Observe(time.Since(start).Seconds())

		d := domain.FromRequest(r)
		fileServed := d.ServeFileHTTP(w, r)

		if !fileServed {
			// We need to trigger authentication flow here if file does not exist to prevent exposing possibly private project existence,
			// because the projects override the paths of the namespace project and they might be private even though
			// namespace project is public
			if d.IsNamespaceProject(r) {
				if auth.CheckAuthenticationWithoutProject(w, r, d) {
					return
				}
			}

			// d found and authentication succeeds
			d.ServeNotFoundHTTP(w, r)
		}
	})
}