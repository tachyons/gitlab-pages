package local

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestDisk_ServeFileHTTP(t *testing.T) {
	defer setUpTests(t)()

	tests := map[string]struct {
		vfsPath        string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		"accessing /index.html": {
			vfsPath:        "group/serving/public",
			path:           "/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "HTML Document",
		},
		"accessing /": {
			vfsPath:        "group/serving/public",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "HTML Document",
		},
		"accessing without /": {
			vfsPath:        "group/serving/public",
			path:           "",
			expectedStatus: http.StatusFound,
			expectedBody:   `<a href="//group.gitlab-example.com/serving/">Found</a>.`,
		},
		"accessing vfs path that is missing": {
			vfsPath: "group/serving/public-missing",
			path:    "/index.html",
			// we expect the status to not be set
			expectedStatus: 0,
		},
		"accessing vfs path that is forbidden (like file)": {
			vfsPath:        "group/serving/public/index.html",
			path:           "/index.html",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	s := Instance()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			w.Code = 0 // ensure that code is not set, and it is being set by handler
			r := httptest.NewRequest("GET", "http://group.gitlab-example.com/serving"+test.path, nil)

			handler := serving.Handler{
				Writer:  w,
				Request: r,
				LookupPath: &serving.LookupPath{
					Prefix: "/serving/",
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
			testhelpers.Close(t, resp.Body)

			require.Equal(t, test.expectedStatus, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), test.expectedBody)
		})
	}
}

var chdirSet = false

func setUpTests(t testing.TB) func() {
	t.Helper()
	return testhelpers.ChdirInPath(t, "../../../../shared/pages", &chdirSet)
}
