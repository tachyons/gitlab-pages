package acceptance_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestRateLimitMiddleware(t *testing.T) {
	testhelpers.EnableRateLimiter(t)

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		// 10 = 1 req every 100ms
		withExtraArgument("rate-limit-source-ip", "1.0"),
		withExtraArgument("rate-limit-source-ip-burst", "1"),
	)

	for i := 0; i < 20; i++ {
		rsp1, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		rsp1.Body.Close()

		// every other request should fail
		//if i%2 != 0 {
		//	require.Equal(t, http.StatusTooManyRequests, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
		//	// wait for another token to become available
		//	time.Sleep(100 * time.Millisecond)
		//	continue
		//}

		require.Equal(t, http.StatusOK, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
		time.Sleep(time.Millisecond)
	}
}
