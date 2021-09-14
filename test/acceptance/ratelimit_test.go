package acceptance_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		//refills 1 token every 50ms, bound by the burst/bucket size
		withExtraArgument("req-domain-per-second", "50ms"),
		// allows a max of 10 tokens at a time PER SECOND
		withExtraArgument("req-domain-bucket-size", "10"),
	)

	for i := 0; i < 20; i++ {
		rsp1, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		defer rsp1.Body.Close()
		fmt.Printf("req: %d - status: %d\n", i, rsp1.StatusCode)

		// every ~10th request should fail
		if (i+1)%10 == 0 {
			require.Equal(t, http.StatusTooManyRequests, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		require.Equal(t, http.StatusOK, rsp1.StatusCode, "group.gitlab-example.com request: %d failed", i)
		// sleep almost close to req-domain-per-second
		time.Sleep(49 * time.Millisecond)
	}
}
