package ratelimiter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainRateLimiter(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	enableRateLimiter(t)

	for tn, tc := range sharedTestCases {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithPerDomainFrequency(tc.domainRate),
				WithPerDomainBurstSize(tc.perDomainBurstPerSecond),
			)

			for i := 0; i < tc.reqNum; i++ {
				ww := httptest.NewRecorder()
				rr := httptest.NewRequest(http.MethodGet, "http://domain.gitlab.io", nil)
				handler := DomainRateLimiter(rl)(next)

				handler.ServeHTTP(ww, rr)
				res := ww.Result()

				if i < tc.perDomainBurstPerSecond {
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

func enableRateLimiter(t *testing.T) {
	t.Helper()

	orig := os.Getenv("FF_ENABLE_RATE_LIMITER")

	err := os.Setenv("FF_ENABLE_RATE_LIMITER", "true")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Setenv("FF_ENABLE_RATE_LIMITER", orig)
	})
}
