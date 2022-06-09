package healthcheck_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/healthcheck"
)

func TestHealthCheckHandler(t *testing.T) {
	u := "https://example.com/-/healthcheck"

	require.HTTPStatusCode(t, healthcheck.Handler().ServeHTTP, http.MethodGet, u, nil, http.StatusOK)
	require.HTTPBodyContains(t, healthcheck.Handler().ServeHTTP, http.MethodGet, u, nil, "success\n")
}
