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

func TestDomainAllowed(t *testing.T) {
	t.Parallel()

	tcs := map[string]struct {
		now                     string
		domainRate              time.Duration
		perDomainBurstPerSecond int
		reqNum                  int
	}{
		"one_request_per_nanosecond": {
			domainRate:              time.Nanosecond, // 1 per nanosecond
			perDomainBurstPerSecond: 1,
			reqNum:                  2,
		},
		"one_request_per_nanosecond_but_big_bucket": {
			domainRate:              time.Nanosecond,
			perDomainBurstPerSecond: 10,
			reqNum:                  11,
		},
		"three_req_per_second_bucket_size_one": {
			domainRate:              3, // 3 per second
			perDomainBurstPerSecond: 1, // max burst 1 means 1 at a time
			reqNum:                  3,
		},
		"10_requests_per_second": {
			domainRate:              10,
			perDomainBurstPerSecond: 10,
			reqNum:                  11,
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithPerDomainFrequency(tc.domainRate),
				WithPerDomainBurstSize(tc.perDomainBurstPerSecond),
			)

			for i := 0; i < tc.reqNum; i++ {
				got := rl.DomainAllowed("rate.gitlab.io")
				if i < tc.perDomainBurstPerSecond {
					require.Truef(t, got, "expected true for request no. %d", i)
				} else {
					// requests should fail after reaching tc.perDomainBurstPerSecond because mockNow
					// always returns the same time
					require.False(t, got, "expected false for request no. %d", i)
				}
			}
		})
	}
}

func TestSingleRateLimiterWithMultipleDomains(t *testing.T) {
	rate := 10 * time.Millisecond
	rl := New(
		WithPerDomainFrequency(rate),
		WithPerDomainBurstSize(1),
	)

	wg := sync.WaitGroup{}
	wg.Add(3)

	testFn := func(domain string) func(t *testing.T) {
		return func(t *testing.T) {
			go func() {
				defer wg.Done()

				for i := 0; i < 5; i++ {
					got := rl.DomainAllowed(domain)
					require.Truef(t, got, "expected true for request no. %d", i)
					time.Sleep(rate)
				}
			}()
		}
	}

	first := "first.gitlab.io"
	t.Run(first, testFn(first))

	second := "second.gitlab.io"
	t.Run(second, testFn(second))

	third := "third.gitlab.io"
	t.Run(third, testFn(third))

	wg.Wait()
}
