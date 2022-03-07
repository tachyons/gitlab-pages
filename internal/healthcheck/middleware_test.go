package healthcheck_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/healthcheck"
)

func TestHealthCheckMiddleware(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "Not a healthcheck request",
			path: "/foo/bar",
			body: "Hello from inner handler",
		},
		{
			name: "Healthcheck request",
			path: "/-/healthcheck",
			body: "success\n",
		},
	}

	cfg := config.Config{
		General: config.General{
			StatusPath: "/-/healthcheck",
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Hello from inner handler")
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()

			middleware := healthcheck.NewMiddleware(handler, cfg.General.StatusPath)
			middleware.ServeHTTP(rr, r)

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, tc.body, rr.Body.String())
		})
	}
}
