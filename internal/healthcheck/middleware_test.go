package healthcheck_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/healthcheck"
)

func TestHealthCheckMiddleware(t *testing.T) {
	tests := map[string]struct {
		path string
		body string
	}{
		"Not a healthcheck request": {
			path: "/foo/bar",
			body: "Hello from inner handler",
		},
		"Healthcheck request": {
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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			middleware := healthcheck.NewMiddleware(handler, cfg.General.StatusPath)

			u := "https://example.com" + tc.path

			require.HTTPStatusCode(t, middleware.ServeHTTP, http.MethodGet, u, nil, http.StatusOK)
			require.HTTPBodyContains(t, middleware.ServeHTTP, http.MethodGet, u, nil, tc.body)
		})
	}
}
