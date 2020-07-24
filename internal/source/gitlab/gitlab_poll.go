package gitlab

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// DefaultPollingMaxRetries to be used by Poll
	DefaultPollingMaxRetries = 30
	// DefaultPollingInterval to be used by Poll
	DefaultPollingInterval = 60 * time.Second
)

// Poll tries to call the /internal/pages/status API endpoint once plus
// `retries` every `interval`.
// TODO: Remove in https://gitlab.com/gitlab-org/gitlab/-/issues/218357
func (g *Gitlab) Poll(retries int, interval time.Duration) error {
	var err error
	for i := 0; i <= retries; i++ {
		log.Info("polling GitLab internal pages status API")
		err = g.checker.Status()
		if err == nil {
			log.Info("GitLab internal pages status API connected successfully")

			// return as soon as we connect to the API
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("polling failed after %d tries every %fs: %w", retries+1, interval.Seconds(), err)
}
