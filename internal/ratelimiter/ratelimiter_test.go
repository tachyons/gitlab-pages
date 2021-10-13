package ratelimiter

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	now          = "2021-09-13T15:00:00Z"
	validTime, _ = time.Parse(time.RFC3339, now)
)

func mockNow() time.Time {
	return validTime
}

var sharedTestCases = map[string]struct {
	sourceIPLimit     float64
	sourceIPBurstSize int
	reqNum            int
}{
	"one_request_per_second": {
		sourceIPLimit:     1,
		sourceIPBurstSize: 1,
		reqNum:            2,
	},
	"one_request_per_second_but_big_bucket": {
		sourceIPLimit:     1,
		sourceIPBurstSize: 10,
		reqNum:            11,
	},
	"three_req_per_second_bucket_size_one": {
		sourceIPLimit:     3,
		sourceIPBurstSize: 1, // max burst 1 means 1 at a time
		reqNum:            3,
	},
	"10_requests_per_second": {
		sourceIPLimit:     10,
		sourceIPBurstSize: 10,
		reqNum:            11,
	},
}

func TestSourceIPAllowed(t *testing.T) {
	t.Parallel()

	for tn, tc := range sharedTestCases {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithSourceIPLimitPerSecond(tc.sourceIPLimit),
				WithSourceIPBurstSize(tc.sourceIPBurstSize),
			)

			for i := 0; i < tc.reqNum; i++ {
				got := rl.SourceIPAllowed("172.16.123.1")
				if i < tc.sourceIPBurstSize {
					require.Truef(t, got, "expected true for request no. %d", i)
				} else {
					// requests should fail after reaching tc.sourceIPBurstSize because mockNow
					// always returns the same time
					require.False(t, got, "expected false for request no. %d", i)
				}
			}
		})
	}
}

func TestSingleRateLimiterWithMultipleSourceIPs(t *testing.T) {
	rate := 10 * time.Millisecond
	rl := New(
		WithSourceIPLimitPerSecond(float64(1/rate)),
		WithSourceIPBurstSize(1),
	)

	wg := sync.WaitGroup{}

	testFn := func(domain string) func(t *testing.T) {
		return func(t *testing.T) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 5; i++ {
					got := rl.SourceIPAllowed(domain)
					require.Truef(t, got, "expected true for request no. %d", i)
					time.Sleep(rate)
				}
			}()
		}
	}

	first := "172.16.123.10"
	t.Run(first, testFn(first))

	second := "172.16.123.20"
	t.Run(second, testFn(second))

	third := "172.16.123.30"
	t.Run(third, testFn(third))

	wg.Wait()
}
