package client

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// DefaultPollingMaxRetries to be used by Poll
	DefaultPollingMaxRetries = 10
	// DefaultPollingInterval to be used by Poll
	DefaultPollingInterval = 10 * time.Second
)

// Poll tries to call the /internal/pages/status API endpoint once plus
// `retries` every `interval`.
// TODO: should we consider using an exponential back-off approach?
// https://pkg.go.dev/github.com/cenkalti/backoff/v4?tab=doc#pkg-examples
func (gc *Client) Poll(retries int, interval time.Duration, errCh chan error) {
	defer close(errCh)

	var err error
	for i := 0; i <= retries; i++ {
		log.Info("polling GitLab internal pages status API")
		err = gc.Status()
		if err == nil {
			log.Info("GitLab internal pages status API connected successfully")

			// return as soon as we connect to the API
			errCh <- nil
			return
		}

		time.Sleep(interval)
	}

	errCh <- fmt.Errorf("polling failed after %d tries every %fs: %w", retries, interval.Seconds(), err)
}
