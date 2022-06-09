package healthcheck

import "net/http"

// Handler is serving the application status check
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte("success\n"))
	})
}
