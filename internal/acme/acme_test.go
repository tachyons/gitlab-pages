package acme_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
)

const (
	baseURL      = "http://example.com"
	indexURL     = baseURL + "/index.html"
	challengeURL = baseURL + "/.well-known/acme-challenge/token"
)

func TestAcmeMiddleware(t *testing.T) {
	u, err := url.Parse("https://gitlab.example.com")
	require.NoError(t, err)

	testCases := []struct {
		name           string
		f              acme.FallbackStrategy
		path           string
		expectedStatus int
	}{
		{
			name:           "not an acme request",
			path:           indexURL,
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "acme challenge redirect to gitlab if missing",
			f: func(w http.ResponseWriter, r *http.Request) bool {
				return false
			},
			path:           challengeURL,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name: "acme challenge served from disk if present",
			f: func(w http.ResponseWriter, r *http.Request) bool {
				w.WriteHeader(http.StatusOK)
				return true
			},
			path:           challengeURL,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !acme.ServeAcmeChallenges(w, r, tc.f, u) {
					w.WriteHeader(http.StatusAccepted)
				}
			})

			require.HTTPStatusCode(t, h.ServeHTTP, http.MethodGet, tc.path, nil, tc.expectedStatus)
		})
	}
}
