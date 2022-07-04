package customheaders

import (
	"net/http"
)

// NewMiddleware returns middleware which inject custom headers into the response
func NewMiddleware(handler http.Handler, headers http.Header) http.Handler {
	if len(headers) == 0 {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddCustomHeaders(w, headers)

		handler.ServeHTTP(w, r)
	})
}
