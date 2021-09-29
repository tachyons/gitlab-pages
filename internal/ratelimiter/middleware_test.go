package ratelimiter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestSourceIPLimiter(t *testing.T) {

	enableRateLimiter(t)

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
				rr.RemoteAddr = "172.16.123.1"

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
				}
			}
		})
	}
}

func TestSourceIPRateLimit(t *testing.T) {
	rl := New(
		WithNow(mockNow),
		WithSourceIPLimitPerSecond(1),
		WithSourceIPBurstSize(1),
	)

	tcs := map[string]struct {
		enabled        bool
		ip             string
		host           string
		expectedStatus int
	}{
		"disabled_rate_limit_http": {
			enabled:        false,
			ip:             "172.16.123.1",
			host:           "http://gitlab.com",
			expectedStatus: http.StatusNoContent,
		},
		"disabled_rate_limit_https": {
			enabled:        false,
			ip:             "172.16.123.2",
			host:           "https://gitlab.com",
			expectedStatus: http.StatusNoContent,
		},
		"enabled_rate_limit_http_does_not_block": {
			enabled:        true,
			ip:             "172.16.123.3",
			host:           "http://gitlab.com",
			expectedStatus: http.StatusNoContent,
		},
		"enabled_rate_limit_https_blocks": {
			enabled:        true,
			ip:             "172.16.123.4",
			host:           "https://gitlab.com",
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {

			for i := 0; i < 5; i++ {
				ww := httptest.NewRecorder()
				rr := httptest.NewRequest(http.MethodGet, tc.host, nil)
				rr.RemoteAddr = tc.ip

				if tc.enabled {
					enableRateLimiter(t)
				}

				handler := rl.SourceIPLimiter(next)

				handler.ServeHTTP(ww, rr)
				res := ww.Result()

				if i == 0 {
					require.Equal(t, http.StatusNoContent, res.StatusCode)
					continue
				}
				// burst is 1 and limit is 1 per second, all subsequent requests should fail
				require.Equal(t, tc.expectedStatus, res.StatusCode)
			}
		})
	}
}

func enableRateLimiter(t *testing.T) {
	t.Helper()

	orig := os.Getenv("FF_ENABLE_RATE_LIMITER")

	err := os.Setenv("FF_ENABLE_RATE_LIMITER", "true")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Setenv("FF_ENABLE_RATE_LIMITER", orig)
	})
}
