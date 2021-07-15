package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisabledRedirects(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withListeners([]ListenSpec{httpListener}),
		withEnv([]string{"FF_ENABLE_REDIRECTS=false"}),
	)

	// Test that redirects status page is forbidden
	rsp, err := GetPageFromListener(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/_redirects")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusForbidden, rsp.StatusCode)

	// Test that redirects are disabled
	rsp, err = GetRedirectPage(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/redirect-portal.html")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestRedirectStatusPage(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withListeners([]ListenSpec{httpListener}),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/_redirects")
	require.NoError(t, err)

	body, err := ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Contains(t, string(body), "11 rules")
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestRedirect(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withListeners([]ListenSpec{httpListener}),
	)

	// Test that serving a file still works with redirects enabled
	rsp, err := GetRedirectPage(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusOK, rsp.StatusCode)

	tests := []struct {
		host             string
		path             string
		expectedStatus   int
		expectedLocation string
	}{
		// Project domain
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/project-redirects/magic-land.html",
		},
		// Make sure invalid rule does not redirect
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/goto-domain.html",
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		// Actual file on disk should override any redirects that match
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/file-override.html",
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
		},
		// Group-level domain
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/magic-land.html",
		},
		// Custom domain
		{
			host:             "redirects.custom-domain.com",
			path:             "/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/magic-land.html",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s%s -> %s (%d)", tt.host, tt.path, tt.expectedLocation, tt.expectedStatus), func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.host, tt.path)
			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, tt.expectedLocation, rsp.Header.Get("Location"))
			require.Equal(t, tt.expectedStatus, rsp.StatusCode)
		})
	}
}
