package zip

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

	tests := map[string]struct {
		vfsPath        string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		"accessing /index.html": {
			vfsPath:        testServerURL + "/public.zip",
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /": {
			vfsPath:        testServerURL + "/public.zip",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing without /": {
			vfsPath:        testServerURL + "/public.zip",
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   `<a href="//zip.gitlab.io/zip/">Found</a>.`,
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
	}

	config.Default.Zip = &config.ZipServing{
		ExpirationInterval: 10 * time.Second,
		CleanupInterval:    5 * time.Second,
		RefreshInterval:    5 * time.Second,
		OpenTimeout:        5 * time.Second,
	}
	s := Instance()

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
			body, err := ioutil.ReadAll(resp.Body)
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
