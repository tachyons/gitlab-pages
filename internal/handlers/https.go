package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

func HTTPSRedirectMiddleware(handler http.Handler, redirect bool) http.Handler {
	if !redirect {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !request.IsHTTPS(r) {
			redirectToHTTPS(w, r, http.StatusTemporaryRedirect)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = request.SchemeHTTPS
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}
