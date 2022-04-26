package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
)

func TestHTTPSMiddleware(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	httpsURL := "https://example.com"
	httpURL := "http://example.com"

	testCases := map[string]struct {
		redirect         bool
		path             string
		expectedStatus   int
		expectedLocation string
	}{
		"http redirects to https with redirect enabled": {
			redirect:         true,
			path:             httpURL,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: httpsURL,
		},
		"https handled successfully with redirect enabled": {
			redirect:       true,
			path:           httpsURL,
			expectedStatus: http.StatusAccepted,
		},
		"http does not redirect to https with redirect disabled": {
			redirect:       false,
			path:           httpURL,
			expectedStatus: http.StatusAccepted,
		},
		"https handled successfully with redirect disabled": {
			redirect:       false,
			path:           httpsURL,
			expectedStatus: http.StatusAccepted,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			m := handlers.HTTPSRedirectMiddleware(h, tc.redirect)
			require.HTTPStatusCode(t, m.ServeHTTP, http.MethodGet, tc.path, nil, tc.expectedStatus)

			// if we expected a redirect make sure the location header is correct
			if tc.expectedStatus == http.StatusTemporaryRedirect {
				w := httptest.NewRecorder()
				req, err := http.NewRequest(http.MethodGet, tc.path, nil)
				require.NoError(t, err)

				m.ServeHTTP(w, req)

				require.Equal(t, []string{httpsURL}, w.Result().Header["Location"])
			}
		})
	}
}
