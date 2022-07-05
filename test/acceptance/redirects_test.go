package acceptance_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestRedirectStatusPage(t *testing.T) {
	t.Setenv(feature.RedirectsPlaceholders.EnvVariable, "true")

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/_redirects")
	require.NoError(t, err)

	body, err := io.ReadAll(rsp.Body)
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	require.Contains(t, string(body), "14 rules")
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestRedirect(t *testing.T) {
	t.Setenv(feature.RedirectsPlaceholders.EnvVariable, "true")

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	// Test that serving a file still works with redirects enabled
	rsp, err := GetRedirectPage(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/index.html")
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

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
		// Permanent redirect for splat (*) with replacement (:splat)
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/jobs/assistant-to-the-regional-manager.html",
			expectedStatus:   http.StatusMovedPermanently,
			expectedLocation: "/project-redirects/careers/assistant-to-the-regional-manager.html",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s%s -> %s (%d)", tt.host, tt.path, tt.expectedLocation, tt.expectedStatus), func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.host, tt.path)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, tt.expectedLocation, rsp.Header.Get("Location"))
			require.Equal(t, tt.expectedStatus, rsp.StatusCode)
		})
	}
}
