package healthcheck

import (
	"net/http"
)

// NewMiddleware is serving the application status check
func NewMiddleware(handler http.Handler, statusPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == statusPath {
			w.Header().Set("Cache-Control", "no-store")
			w.Write([]byte("success\n"))

			return
		}

		handler.ServeHTTP(w, r)
	})
}
