package zip

import (
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
		sha256         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		"accessing /index.html": {
			vfsPath:        httpURL,
			sha256:         "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /index.html from disk": {
			vfsPath:        fileURL,
			sha256:         "15c5438164ec67bb2225f68d7d7a2e0b608035264e5275b7e3302641aa25a528",
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /": {
			vfsPath:        httpURL,
			sha256:         "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing / from disk": {
			vfsPath:        fileURL,
			sha256:         "15c5438164ec67bb2225f68d7d7a2e0b608035264e5275b7e3302641aa25a528",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing without /": {
			vfsPath:        httpURL,
			sha256:         "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   `<a href="//zip.gitlab.io/zip/">Found</a>.`,
		},
		"accessing without / from disk": {
			vfsPath:        fileURL,
			sha256:         "15c5438164ec67bb2225f68d7d7a2e0b608035264e5275b7e3302641aa25a528",
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   `<a href="//zip.gitlab.io/zip/">Found</a>.`,
		},
		"accessing archive that is 404": {
			vfsPath: testServerURL + "/invalid.zip",
			// the sha is needed or we would get a 500
			sha256: "foo",
			path:   "/index.html",
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
			r := httptest.NewRequest("GET", "http://zip.gitlab.io/zip"+test.path, nil)

			handler := serving.Handler{
				Writer:  w,
				Request: r,
				LookupPath: &serving.LookupPath{
					Prefix: "/zip/",
					Path:   test.vfsPath,
					SHA256: test.sha256,
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
			defer resp.Body.Close()

			require.Equal(t, test.expectedStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), test.expectedBody)
		})
	}
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
