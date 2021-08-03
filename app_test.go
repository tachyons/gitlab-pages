package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func Test_setRequestScheme(t *testing.T) {
	tests := []struct {
		name           string
		r              *http.Request
		expectedScheme string
	}{
		{
			name:           "http",
			r:              newGetRequestWithScheme(t, request.SchemeHTTP, false),
			expectedScheme: request.SchemeHTTP,
		},
		{
			name:           "https",
			r:              newGetRequestWithScheme(t, request.SchemeHTTPS, true),
			expectedScheme: request.SchemeHTTPS,
		},
		{
			name:           "empty_scheme_no_tls",
			r:              newGetRequestWithScheme(t, "", false),
			expectedScheme: request.SchemeHTTP,
		},
		{
			name:           "empty_scheme_with_tls",
			r:              newGetRequestWithScheme(t, "", true),
			expectedScheme: request.SchemeHTTPS,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setRequestScheme(tt.r)
			require.Equal(t, got.URL.Scheme, tt.expectedScheme)
		})
	}
}

func newGetRequestWithScheme(t *testing.T, scheme string, withTLS bool) *http.Request {
	t.Helper()

	req, err := http.NewRequest("GET", fmt.Sprintf("%s//localost/", scheme), nil)
	require.NoError(t, err)
	req.URL.Scheme = scheme
	if withTLS {
		req.TLS = &tls.ConnectionState{}
	}

	return req
}

func TestHealthCheckMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
		body   string
	}{
		{
			name:   "Not a healthcheck request",
			path:   "/foo/bar",
			status: http.StatusOK,
			body:   "Hello from inner handler",
		},
		{
			name:   "Healthcheck request",
			path:   "/-/healthcheck",
			status: http.StatusServiceUnavailable,
			body:   "not yet ready\n",
		},
	}

	// required by LoadConfig
	require.NoError(t, os.Setenv("LISTEN_HTTP", ":0"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("LISTEN_HTTP"))
	})

	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	cfg.General.StatusPath = "/-/healthcheck"

	domains, err := source.NewDomains("auto", &cfg.GitLab)
	require.NoError(t, err)

	app := theApp{
		config:  cfg,
		domains: domains,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Hello from inner handler")
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			middleware, err := app.healthCheckMiddleware(handler)
			require.NoError(t, err)
			middleware.ServeHTTP(rr, r)

			require.Equal(t, tc.status, rr.Code)
			require.Equal(t, tc.body, rr.Body.String())
		})
	}
}
