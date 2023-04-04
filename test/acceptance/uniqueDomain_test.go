package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestRedirectToUniqueDomain(t *testing.T) {
	RunPagesProcess(t)

	tests := []struct {
		name          string
		requestDomain string
		requestPath   string
		redirectURL   string
		httpStatus    int
	}{
		{
			name:          "when project has unique domain",
			requestDomain: "group.unique-url.gitlab-example.com",
			requestPath:   "with-unique-url/",
			redirectURL:   "https://unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com/",
			httpStatus:    http.StatusPermanentRedirect,
		},
		{
			name:          "when requesting implicit index.html",
			requestDomain: "group.unique-url.gitlab-example.com",
			requestPath:   "with-unique-url",
			redirectURL:   "https://unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			httpStatus:    http.StatusPermanentRedirect,
		},
		{
			name:          "when project is nested",
			requestDomain: "group.unique-url.gitlab-example.com",
			requestPath:   "subgroup1/subgroup2/with-unique-url/subdir/index.html",
			redirectURL:   "https://unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com/subdir/index.html",
			httpStatus:    http.StatusPermanentRedirect,
		},
		{
			name:          "when serving with a port",
			requestDomain: "group.unique-url.gitlab-example.com:8080",
			requestPath:   "with-unique-url/",
			redirectURL:   "https://unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com:8080/",
			httpStatus:    http.StatusPermanentRedirect,
		},
		{
			name:          "when serving a path with a port",
			requestDomain: "group.unique-url.gitlab-example.com:8080",
			requestPath:   "with-unique-url/subdir/index.html",
			redirectURL:   "https://unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com:8080/subdir/index.html",
			httpStatus:    http.StatusPermanentRedirect,
		},
		{
			name:          "when already serving the unique domain",
			requestDomain: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "when already serving the unique domain with a port",
			requestDomain: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com:8080",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "when already serving the unique domain with path",
			requestDomain: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			requestPath:   "subdir/index.html",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "when project does not have unique domain with a path",
			requestDomain: "group.unique-url.gitlab-example.com",
			requestPath:   "without-unique-url/subdir/index.html",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "when project does not exist",
			requestDomain: "group.unique-url.gitlab-example.com",
			requestPath:   "inexisting-project/",
			httpStatus:    http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpsListener, test.requestDomain, test.requestPath)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, test.httpStatus, rsp.StatusCode)
			require.Equal(t, test.redirectURL, rsp.Header.Get("Location"))
		})
	}
}
