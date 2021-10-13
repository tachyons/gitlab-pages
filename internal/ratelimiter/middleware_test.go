package ratelimiter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

const (
	remoteAddr = "192.168.1.1"
)

var next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestSourceIPLimiterWithDifferentLimits(t *testing.T) {
	hook := testlog.NewGlobal()
	testhelpers.SetEnvironmentVariable(t, testhelpers.FFEnableRateLimiter, "true")

	for tn, tc := range sharedTestCases {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithSourceIPLimitPerSecond(tc.sourceIPLimit),
				WithSourceIPBurstSize(tc.sourceIPBurstSize),
			)

			for i := 0; i < tc.reqNum; i++ {
				ww := httptest.NewRecorder()
				rr := httptest.NewRequest(http.MethodGet, "https://domain.gitlab.io", nil)
				rr.RemoteAddr = remoteAddr

				handler := rl.SourceIPLimiter(next)

				handler.ServeHTTP(ww, rr)
				res := ww.Result()

				if i < tc.sourceIPBurstSize {
					require.Equal(t, http.StatusNoContent, res.StatusCode, "req: %d failed", i)
				} else {
					// requests should fail after reaching tc.perDomainBurstPerSecond because mockNow
					// always returns the same time
					require.Equal(t, http.StatusTooManyRequests, res.StatusCode, "req: %d failed", i)
					b, err := io.ReadAll(res.Body)
					require.NoError(t, err)

					require.Contains(t, string(b), "Too many requests.")
					res.Body.Close()

					assertSourceIPLog(t, remoteAddr, hook)
				}
			}
		})
	}
}

func TestSourceIPLimiterDenyRequestsAfterBurst(t *testing.T) {
	hook := testlog.NewGlobal()

	tcs := map[string]struct {
		enabled        bool
		expectedStatus int
	}{
		"disabled_rate_limit_http": {
			enabled:        false,
			expectedStatus: http.StatusNoContent,
		},
		"disabled_rate_limit_https": {
			enabled:        false,
			expectedStatus: http.StatusNoContent,
		},
		"enabled_rate_limit_http_blocks": {
			enabled:        true,
			expectedStatus: http.StatusTooManyRequests,
		},
		"enabled_rate_limit_https_blocks": {
			enabled:        true,
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithSourceIPLimitPerSecond(1),
				WithSourceIPBurstSize(1),
			)

			for i := 0; i < 5; i++ {
				ww := httptest.NewRecorder()
				rr := httptest.NewRequest(http.MethodGet, "http://gitlab.com", nil)
				if tc.enabled {
					testhelpers.SetEnvironmentVariable(t, testhelpers.FFEnableRateLimiter, "true")
				} else {
					testhelpers.SetEnvironmentVariable(t, testhelpers.FFEnableRateLimiter, "false")
				}

				rr.RemoteAddr = remoteAddr

				// middleware is evaluated in reverse order
				handler := rl.SourceIPLimiter(next)

				handler.ServeHTTP(ww, rr)
				res := ww.Result()

				if i == 0 {
					require.Equal(t, http.StatusNoContent, res.StatusCode)
					continue
				}

				// burst is 1 and limit is 1 per second, all subsequent requests should fail
				require.Equal(t, tc.expectedStatus, res.StatusCode)
				assertSourceIPLog(t, remoteAddr, hook)
			}
		})
	}
}

func assertSourceIPLog(t *testing.T, remoteAddr string, hook *testlog.Hook) {
	t.Helper()

	require.NotNil(t, hook.LastEntry())

	// source_ip that was rate limited
	require.Equal(t, remoteAddr, hook.LastEntry().Data["source_ip"])

	hook.Reset()
}
