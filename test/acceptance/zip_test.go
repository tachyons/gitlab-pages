package acceptance_test

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestZipServing(t *testing.T) {
	_, cleanup := newZipFileServerURL(t, "../../shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	source := NewGitlabDomainsSourceStub(t, &stubOpts{})
	defer source.Close()

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

	pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", "gitlab"}
	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, supportedListeners(), "", []string{}, pagesArgs...)
	defer teardown()

	tests := map[string]struct {
		host               string
		urlSuffix          string
		expectedStatusCode int
		expectedContent    string
	}{
		"base_domain_no_suffix": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/index.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists_in_subdir": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/subdir/hello.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/subdir/hello.html\n",
		},
		"file_exists_symlink": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/symlink.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "symlink.html->subdir/linked.html\n",
		},
		"dir": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/subdir/",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"file_does_not_exist": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/unknown.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"bad_symlink": {
			host:               "zip.gitlab.io",
			urlSuffix:          "/bad-symlink.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"with_not_found_zip": {
			host:               "zip-not-found.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "The page you're looking for could not be found",
		},
		"with_malformed_zip": {
			host:               "zip-malformed.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusInternalServerError,
			expectedContent:    "Something went wrong (500)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := GetPageFromListener(t, httpListener, tt.host, tt.urlSuffix)
			require.NoError(t, err)
			defer response.Body.Close()

			require.Equal(t, tt.expectedStatusCode, response.StatusCode)

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedContent, "content mismatch")
		})
	}
}

func TestZipServingFromDisk(t *testing.T) {
	chdir := false
	defer testhelpers.ChdirInPath(t, "../../shared/pages", &chdir)()

	_, cleanup := newZipFileServerURL(t, "shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)

	source := NewGitlabDomainsSourceStub(t, &stubOpts{
		pagesRoot: wd,
	})

	defer source.Close()

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

	pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", "gitlab", "-pages-root", wd}
	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, supportedListeners(), "", []string{}, pagesArgs...)
	defer teardown()

	tests := map[string]struct {
		host               string
		urlSuffix          string
		expectedStatusCode int
		expectedContent    string
	}{
		"base_domain_no_suffix": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/index.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists_in_subdir": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/subdir/hello.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/subdir/hello.html\n",
		},
		"file_exists_symlink": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/symlink.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "symlink.html->subdir/linked.html\n",
		},
		"dir": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/subdir/",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"file_does_not_exist": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/unknown.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"bad_symlink": {
			host:               "zip-from-disk.gitlab.io",
			urlSuffix:          "/bad-symlink.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"with_not_found_zip": {
			host:               "zip-from-disk-not-found.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "The page you're looking for could not be found",
		},
		"file_not_allowed_in_path": {
			host:               "zip-not-allowed-path.gitlab.io",
			urlSuffix:          "/",
			expectedStatusCode: http.StatusInternalServerError,
			expectedContent:    "Whoops, something went wrong on our end.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := GetPageFromListener(t, httpListener, tt.host, tt.urlSuffix)
			require.NoError(t, err)
			defer response.Body.Close()

			require.Equal(t, tt.expectedStatusCode, response.StatusCode)

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedContent, "content mismatch")
		})
	}
}

func TestZipServingConfigShortTimeout(t *testing.T) {
	source := NewGitlabDomainsSourceStub(t, &stubOpts{})
	defer source.Close()

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

	pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", "gitlab",
		"-zip-open-timeout=1ns"} // <- test purpose

	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, supportedListeners(), "", []string{}, pagesArgs...)
	defer teardown()

	_, cleanup := newZipFileServerURL(t, "../../shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	response, err := GetPageFromListener(t, httpListener, "zip.gitlab.io", "/")
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusInternalServerError, response.StatusCode, "should fail to serve")
}

func newZipFileServerURL(t *testing.T, zipFilePath string) (string, func()) {
	t.Helper()

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
	}))
	m.HandleFunc("/malformed.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	// create a listener with the desired port.
	l, err := net.Listen("tcp", objectStorageMockServer)
	require.NoError(t, err)

	testServer := httptest.NewUnstartedServer(m)

	// NewUnstartedServer creates a listener. Close that listener and replace
	// with the one we created.
	testServer.Listener.Close()
	testServer.Listener = l

	// Start the server.
	testServer.Start()

	return testServer.URL, func() {
		// Cleanup.
		testServer.Close()
	}
}
