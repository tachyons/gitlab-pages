package urilimiter

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

func NewMiddleware(handler http.Handler, limit int) http.Handler {
	if limit == 0 {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.RequestURI) > limit {
			httperrors.Serve414(w)

			return
		}

		handler.ServeHTTP(w, r)
	})
}
