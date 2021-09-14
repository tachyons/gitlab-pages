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
		withExtraArgument("enable-rate-limiter", "true"),
		//refills 1 token every 10ms, bound by the burst/bucket size
		withExtraArgument("rate-limit-per-domain", "10ms"),
		// allows a max of 1 token at a time PER instance of time
		withExtraArgument("rate-limit-per-domain-burst-size", "1"),
	)

	for i := 0; i < 20; i++ {
		rsp1, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		rsp1.Body.Close()

		// every other request should fail
		if i%2 != 0 {
			require.Equal(t, http.StatusTooManyRequests, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
			// wait for another token to become available
			time.Sleep(10 * time.Millisecond)
			continue
		}

		require.Equal(t, http.StatusOK, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
		time.Sleep(time.Millisecond)
	}
}
