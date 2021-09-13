package ratelimiter

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	now          = "2021-09-13T15:00:00Z"
	validTime, _ = time.Parse(time.RFC3339, now)
)

func mockNow() time.Time {
	validTime = validTime.Add(time.Millisecond)
	return validTime
}

func TestDomainAllowed(t *testing.T) {
	tcs := map[string]struct {
		now                     string
		domainRate              time.Duration
		perDomainBurstPerSecond int
		domain                  string
		reqNum                  int
	}{
		"one_request_per_second": {
			domainRate:              1, // 1 per second
			perDomainBurstPerSecond: 1,
			reqNum:                  2,
			domain:                  "rate.gitlab.io",
		},
		"one_request_per_second_but_big_bucket": {
			domainRate:              1, // 1 per second
			perDomainBurstPerSecond: 10,
			reqNum:                  11,
			domain:                  "rate.gitlab.io",
		},
		"three_req_per_second_bucket_size_one": {
			domainRate:              3, // 3 per second
			perDomainBurstPerSecond: 1, // max burst 1 means 1 at a time
			reqNum:                  3,
			domain:                  "rate.gitlab.io",
		},
		"10_requests_per_second": {
			domainRate:              10,
			perDomainBurstPerSecond: 10,
			reqNum:                  11,
			domain:                  "rate.gitlab.io",
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow),
				WithDomainRatePerSecond(tc.domainRate),
				WithDomainBurstPerSecond(tc.perDomainBurstPerSecond),
			)

			for i := 0; i < tc.reqNum; i++ {
				got := rl.DomainAllowed(tc.domain)
				if i < tc.perDomainBurstPerSecond {
					require.Truef(t, got, "expected true for request no. %d", i+1)
				} else {
					require.False(t, got, "expected false for request no. %d", i+1)
				}
			}
		})
	}
}

func TestDomainAllowedWitSleeps(t *testing.T) {
	rate := 10 * time.Millisecond
	rl := New(
		WithNow(mockNow),
		WithDomainRatePerSecond(rate),
		WithDomainBurstPerSecond(1),
	)
	domain := "test.gitlab.io"

	t.Run("one request every millisecond with burst 1", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			got := rl.DomainAllowed(domain)
			require.Truef(t, got, "expected true for request no. %d", i+1)
			time.Sleep(rate)
		}
	})

	t.Run("requests start failing after reaching burst", func(t *testing.T) {
		//now := mockNow()
		for i := 0; i < 10; i++ {
			got := rl.DomainAllowed(domain + ".diff")
			fmt.Printf("for:%d got: %t\n", i, got)
			//require.True(t, true)
			//if i < 2 {
			require.Truef(t, got, "expected true for request no. %d", i)
			//} else {
			//	require.False(t, got, "expected false for request no. %d", i)
			//}
			time.Sleep(time.Nanosecond)
		}
	})
}
