package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
)

func TestAcmeMiddleware(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	httpsURL := "https://example.com"
	httpURL := "http://example.com"

	testCases := []struct {
		name           string
		redirect       bool
		path           string
		expectedStatus int
	}{
		{
			name:           "http redirects to https with redirect enabled",
			redirect:       true,
			path:           httpURL,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "https handled successfully with redirect enabled",
			redirect:       true,
			path:           httpsURL,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "http does not redirect to https with redirect disabled",
			redirect:       false,
			path:           httpURL,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "https handled successfully with redirect disabled",
			redirect:       false,
			path:           httpsURL,
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := handlers.HTTPSRedirectMiddleware(h, tc.redirect)
			require.HTTPStatusCode(t, m.ServeHTTP, http.MethodGet, tc.path, nil, tc.expectedStatus)
		})
	}
}
