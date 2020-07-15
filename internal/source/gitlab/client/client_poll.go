package client

import (
	"fmt"
	"time"
)

const (
	// DefaultPollingMaxRetries to be used by Poll
	DefaultPollingMaxRetries = 30
	// DefaultPollingInterval to be used by Poll
	DefaultPollingInterval = 10 * time.Second
)

// Poll tries to call the /internal/pages/status API endpoint for
// `retries` every `interval`.
// TODO: should we consider using an exponential back-off approach?
// https://pkg.go.dev/github.com/cenkalti/backoff/v4?tab=doc#pkg-examples
func (gc *Client) Poll(retries int, interval time.Duration, errCh chan error) {
	defer close(errCh)

	var err error
	for i := 0; i <= retries; i++ {
		err = gc.Status()
		if err == nil {
			// return as soon as we connect to the API
			errCh <- nil
		}

		time.Sleep(interval)
	}

	errCh <- fmt.Errorf("polling failed after %d tries every %fs: %w", retries, interval.Seconds(), err)
}
