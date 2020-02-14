package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
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

func TestIsHealthCheckRequest(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		isHealthcheck bool
	}{
		{
			name:          "Not a healthcheck request",
			path:          "/foo/bar",
			isHealthcheck: false,
		},
		{
			name:          "Healthcheck request",
			path:          "/-/healthcheck",
			isHealthcheck: true,
		},
	}

	app := theApp{}
	app.appConfig.StatusPath = "/-/healthcheck"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.path, nil)
			require.Equal(t, tc.isHealthcheck, app.isHealthCheckRequest(r))
		})
	}
}

func TestGetHostAndDomainDoNotLookupDomainForHealthCheckRequest(t *testing.T) {
	app := theApp{}
	app.appConfig.StatusPath = "/-/healthcheck"

	r := httptest.NewRequest("GET", "/-/healthcheck", nil)
	r.Host = "gitlab.com"

	host, domain, err := app.getHostAndDomain(r)

	require.Equal(t, "gitlab.com", host)
	require.Nil(t, domain)
	require.NoError(t, err)
}
