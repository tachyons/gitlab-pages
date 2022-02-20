package zip

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestZip_ServeFileHTTP(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public-without-dirs.zip")
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)

	httpURL := testServerURL + "/public.zip"
	fileURL := "file://" + wd + "/group/zip.gitlab.io/public-without-dirs.zip"

	tests := map[string]struct {
		vfsPath        string
		path           string
		expectedStatus int
		expectedBody   string
		extraHeaders   http.Header
	}{
		"accessing /index.html": {
			vfsPath:        httpURL,
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /index.html from disk": {
			vfsPath:        fileURL,
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing / If-Modified-Since": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusNotModified,
			extraHeaders: http.Header{
				"If-Modified-Since": {time.Now().Format(http.TimeFormat)},
			},
		},
		"accessing / If-Modified-Since fails": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
			extraHeaders: http.Header{
				"If-Modified-Since": {time.Now().AddDate(-10, 0, 0).Format(http.TimeFormat)},
			},
		},
		"accessing / If-Unmodified-Since": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusPreconditionFailed,
			extraHeaders: http.Header{
				"If-Unmodified-Since": {time.Now().AddDate(-10, 0, 0).Format(http.TimeFormat)},
			},
		},
		"accessing / If-Unmodified-Since fails": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
			extraHeaders: http.Header{
				"If-Unmodified-Since": {time.Now().Format(http.TimeFormat)},
			},
		},
		"accessing / from disk": {
			vfsPath:        fileURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing without /": {
			vfsPath:        httpURL,
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   "<a href=\"//zip.gitlab.io/zip/\">Found</a>.\n\n",
		},
		"accessing without / from disk": {
			vfsPath:        fileURL,
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   "<a href=\"//zip.gitlab.io/zip/\">Found</a>.\n\n",
		},
		"accessing archive that is 404": {
			vfsPath: testServerURL + "/invalid.zip",
			path:    "/index.html",
			// we expect the status to not be set
			expectedStatus: 0,
		},
		"accessing archive that is 500": {
			vfsPath:        testServerURL + "/500",
			path:           "/index.html",
			expectedStatus: http.StatusInternalServerError,
		},
		"accessing file:// outside of allowedPaths": {
			vfsPath:        "file:///some/file/outside/path",
			path:           "/index.html",
			expectedStatus: http.StatusInternalServerError,
		},
		"accessing / If-None-Match": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusNotModified,
			extraHeaders: http.Header{
				"If-None-Match": {fmt.Sprintf("%q", sha(httpURL))},
			},
		},
		"accessing / If-None-Match fails": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
			extraHeaders: http.Header{
				"If-None-Match": {fmt.Sprintf("%q", "badetag")},
			},
		},
		"accessing / If-Match": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
			extraHeaders: http.Header{
				"If-Match": {fmt.Sprintf("%q", sha(httpURL))},
			},
		},
		"accessing / If-Match fails": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusPreconditionFailed,
			extraHeaders: http.Header{
				"If-Match": {fmt.Sprintf("%q", "wrongetag")},
			},
		},
		"accessing / If-Match fails2": {
			vfsPath:        httpURL,
			path:           "/",
			expectedStatus: http.StatusPreconditionFailed,
			extraHeaders: http.Header{
				"If-Match": {","},
			},
		},
	}

	cfg := &config.Config{
		Zip: config.ZipServing{
			ExpirationInterval: 10 * time.Second,
			CleanupInterval:    5 * time.Second,
			RefreshInterval:    5 * time.Second,
			OpenTimeout:        5 * time.Second,
			AllowedPaths:       []string{wd},
		},
	}

	s := Instance()
	err = s.Reconfigure(cfg)
	require.NoError(t, err)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			w.Code = 0 // ensure that code is not set, and it is being set by handler
			r := httptest.NewRequest(http.MethodGet, "http://zip.gitlab.io/zip"+test.path, nil)

			if test.extraHeaders != nil {
				r.Header = test.extraHeaders
			}

			handler := serving.Handler{
				Writer:  w,
				Request: r,
				LookupPath: &serving.LookupPath{
					Prefix: "/zip/",
					Path:   test.vfsPath,
					SHA256: sha(test.vfsPath),
				},
				SubPath: test.path,
			}

			if test.expectedStatus == 0 {
				require.False(t, s.ServeFileHTTP(handler))
				require.Zero(t, w.Code, "we expect status to not be set")
				return
			}

			require.True(t, s.ServeFileHTTP(handler))

			resp := w.Result()
			testhelpers.Close(t, resp.Body)

			require.Equal(t, test.expectedStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if test.expectedStatus == http.StatusOK {
				require.NotEmpty(t, resp.Header.Get("Last-Modified"))
				require.NotEmpty(t, resp.Header.Get("ETag"))
			}

			if test.expectedStatus != http.StatusInternalServerError {
				require.Equal(t, test.expectedBody, string(body))
			}
		})
	}
}

func sha(path string) string {
	sha := sha256.Sum256([]byte(path))
	s := hex.EncodeToString(sha[:])
	return s
}

var chdirSet = false

func newZipFileServerURL(t *testing.T, zipFilePath string) (string, func()) {
	t.Helper()

	chdir := testhelpers.ChdirInPath(t, "../../../../shared/pages", &chdirSet)

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
	}))
	m.HandleFunc("/500", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	testServer := httptest.NewServer(m)

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}
