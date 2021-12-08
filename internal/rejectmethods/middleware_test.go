package rejectmethods

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK\n")
	})

	middleware := NewMiddleware(handler)

	acceptedMethods := []string{"GET", "HEAD", "POST", "PUT", "PATCH", "CONNECT", "OPTIONS", "TRACE"}
	for _, method := range acceptedMethods {
		t.Run(method, func(t *testing.T) {
			require.HTTPStatusCode(t, middleware.ServeHTTP, method, "/", nil, http.StatusOK)
		})
	}

	t.Run("UNKNOWN", func(t *testing.T) {
		require.HTTPStatusCode(t, middleware.ServeHTTP, "UNKNOWN", "/", nil, http.StatusMethodNotAllowed)
	})
}
