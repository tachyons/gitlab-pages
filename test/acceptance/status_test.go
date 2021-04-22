package acceptance_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatusPage(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, SupportedListeners(), "", "-pages-status=/@statuscheck")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestStatusNotYetReady(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, SupportedListeners(), "", "-pages-status=/@statuscheck", "-pages-root=../../shared/invalid-pages")
	defer teardown()

	waitForRoundtrips(t, SupportedListeners(), 5*time.Second)
	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestPageNotAvailableIfNotLoaded(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, SupportedListeners(), "", "-pages-root=../../shared/invalid-pages")
	defer teardown()
	waitForRoundtrips(t, SupportedListeners(), 5*time.Second)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}
