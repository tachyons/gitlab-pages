package acceptance_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatusPage(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withListeners([]ListenSpec{httpListener}),
		withExtraArgument("pages-status", "/@statuscheck"),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestStatusNotYetReady(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withoutWait,
		withExtraArgument("pages-status", "/@statuscheck"),
		withExtraArgument("pages-root", "../../shared/invalid-pages"),
		withStubOptions(&stubOpts{
			statusReadyCount: 100,
		}),
	)

	waitForRoundtrips(t, supportedListeners(), 5*time.Second)
	for _, spec := range supportedListeners() {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "@statuscheck")
		require.NoError(t, err)
		defer rsp.Body.Close()
		require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)

		rsp2, err2 := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html")
		require.NoError(t, err2)
		defer rsp2.Body.Close()
		require.Equal(t, http.StatusServiceUnavailable, rsp2.StatusCode, "page should not be served")
	}
}
