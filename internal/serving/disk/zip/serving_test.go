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
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip")
	defer cleanup()

	s := Instance()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://zip.gitlab.io/zip/index.html", nil)
	handler := serving.Handler{
		Writer:  w,
		Request: r,
		LookupPath: &serving.LookupPath{
			Prefix: "",
			Path:   testServerURL + "/public.zip",
		},
		SubPath: "/index.html",
	}

	require.True(t, s.ServeFileHTTP(handler))

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "zip.gitlab.io/project/index.html\n")
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
