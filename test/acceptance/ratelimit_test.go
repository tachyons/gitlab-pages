package acceptance_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		// allows a max of 2 tokens every 10ms -> 2rp 10ms
		withExtraArgument("req-domain-per-second", "100ms"),
		withExtraArgument("req-domain-bucket-size", "2"),
	)

	rsp1, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp1.Body.Close()

	// make another request right away should fail
	rsp2, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// wait for ratelimiter to clear
	time.Sleep(300 * time.Millisecond)

	rsp3, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp3.Body.Close()

	// request another domain
	rsp4, err := GetPageFromListener(t, httpListener, "CapitalGroup.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp4.Body.Close()

	require.Equal(t, http.StatusOK, rsp1.StatusCode, "group.gitlab-example.com first request")
	require.Equal(t, http.StatusTooManyRequests, rsp2.StatusCode, "group.gitlab-example.com without waiting")
	require.Equal(t, http.StatusOK, rsp3.StatusCode, "rsp3 group.gitlab-example.com after waiting 1s")
	require.Equal(t, http.StatusOK, rsp4.StatusCode, "CapitalGroup.gitlab-example.com for another domain")
}
