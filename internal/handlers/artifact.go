package handlers

import "net/http"

func ArtifactMiddleware(handler http.Handler, h *Handlers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.HandleArtifactRequest(w, r) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}
