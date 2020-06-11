package disk

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

func TestDisk_ServeFileHTTP(t *testing.T) {
	s := New()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://group.gitlab-example.com/serving/index.html", nil)
	handler := serving.Handler{
		Writer:  w,
		Request: r,
		LookupPath: &serving.LookupPath{
			Prefix: "/serving",
			Path:   "../../../shared/pages/group/serving/public",
		},
		SubPath: "/index.html",
	}

	require.True(t, s.ServeFileHTTP(handler))

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "HTML Document")
}
