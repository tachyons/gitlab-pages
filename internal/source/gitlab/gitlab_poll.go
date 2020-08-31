package gitlab

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
)

const (
	// maxPollingTime is the maximum duration to try to call the Status API
	maxPollingTime = 60 * time.Minute
)

// Poll tries to call the /internal/pages/status API endpoint once plus
// for `maxElapsedTime`
// TODO: Remove in https://gitlab.com/gitlab-org/gitlab-pages/-/issues/449
func (g *Gitlab) poll(interval, maxElapsedTime time.Duration) {
	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = interval
	backOff.MaxElapsedTime = maxElapsedTime

	operation := func() error {
		log.Info("Checking GitLab internal API availability")

		return g.client.Status()
	}

	err := backoff.Retry(operation, backOff)
	if err != nil {
		log.WithError(err).Errorf("Failed to connect to the internal GitLab API after %.2fs", maxElapsedTime.Seconds())
		return
	}

	g.mu.Lock()
	g.isReady = true
	g.mu.Unlock()

	log.Info("GitLab internal pages status API connected successfully")
}
