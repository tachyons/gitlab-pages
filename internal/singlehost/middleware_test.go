package singlehost

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

var writeURLhandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, r.Host+r.URL.Path)
})

func TestServeHTTP(t *testing.T) {
	handler := NewMiddleware(writeURLhandler, "pages.example.com")

	tests := []struct {
		name        string
		URL         string
		expectedURL string
	}{
		{
			name:        "custom domain",
			URL:         "http://mydomain.example.com",
			expectedURL: "mydomain.example.com",
		},
		{
			name:        "namespace root",
			URL:         "http://pages.example.com/group",
			expectedURL: "group.pages.example.com/",
		},
		{
			name:        "namespace root with port",
			URL:         "http://pages.example.com:8080/group",
			expectedURL: "group.pages.example.com:8080/",
		},
		{
			name:        "namespace root with trailing slash",
			URL:         "http://pages.example.com/group/",
			expectedURL: "group.pages.example.com/",
		},
		{
			name:        "namespace with path",
			URL:         "http://pages.example.com/group/path/to/file",
			expectedURL: "group.pages.example.com/path/to/file",
		},
		{
			name:        "namespace with path does not remove trailing slash",
			URL:         "http://pages.example.com/group/path/to/file/",
			expectedURL: "group.pages.example.com/path/to/file/",
		},
		{
			name:        "namespace with path and port",
			URL:         "http://pages.example.com:8080/group/path/to/file",
			expectedURL: "group.pages.example.com:8080/path/to/file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.URL, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			body, err := ioutil.ReadAll(recorder.Body)
			require.NoError(t, err)

			require.Equal(t, tt.expectedURL, string(body))
		})
	}
}

func TestServeHTTPWithRedirect(t *testing.T) {
	tests := []struct {
		name                string
		redirectURL         string
		expectedRedirectURL string
	}{
		{
			name:                "redirecting to non-group domain",
			redirectURL:         "//example.com:8080/test",
			expectedRedirectURL: "//example.com:8080/test",
		},
		{
			name:                "redirecting to group domain",
			redirectURL:         "//group.pages.example.com:8080/test",
			expectedRedirectURL: "//pages.example.com:8080/group/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redirectHandler := http.RedirectHandler(tt.redirectURL, 302)
			handler := NewMiddleware(redirectHandler, "pages.example.com")

			testhelpers.AssertRedirectTo(t, handler.ServeHTTP, "GET", "/", nil, tt.expectedRedirectURL)
		})
	}
}
