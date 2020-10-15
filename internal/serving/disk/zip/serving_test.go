package zip

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestZip_ServeFileHTTP(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public-without-dirs.zip")
	defer cleanup()

	tests := map[string]struct {
		path           string
		expectedStatus int
		expectedBody   string
	}{
		"accessing /index.html": {
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing /": {
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "zip.gitlab.io/project/index.html\n",
		},
		"accessing without /": {
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   `<a href="//zip.gitlab.io/zip/">Found</a>.`,
		},
	}

	s := Instance()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "http://zip.gitlab.io/zip"+test.path, nil)

			handler := serving.Handler{
				Writer:  w,
				Request: r,
				LookupPath: &serving.LookupPath{
					Prefix: "/zip/",
					Path:   testServerURL + "/public.zip",
				},
				SubPath: test.path,
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

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
	}))

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}
