package testhelpers

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertHTTP404 asserts handler returns 404 with provided str body
func AssertHTTP404(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, str interface{}) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	handler(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, "HTTP status")

	if str != nil {
		contentType, _, _ := mime.ParseMediaType(w.Header().Get("Content-Type"))
		require.Equal(t, "text/html", contentType, "Content-Type")
		require.Contains(t, w.Body.String(), str)
	}
}

// AssertRedirectTo asserts that handler redirects to particular URL
func AssertRedirectTo(t *testing.T, handler http.HandlerFunc, method string,
	url string, values url.Values, expectedURL string) {

	require.HTTPRedirect(t, handler, method, url, values)

	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest(method, url, nil)
	req.URL.RawQuery = values.Encode()

	handler(recorder, req)

	require.Equal(t, expectedURL, recorder.Header().Get("Location"))
}
