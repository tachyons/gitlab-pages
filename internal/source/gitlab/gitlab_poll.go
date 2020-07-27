package gitlab

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// defaultPollingMaxRetries to be used by Poll
	defaultPollingMaxRetries = 30
	// defaultPollingInterval to be used by Poll
	defaultPollingInterval = 10 * time.Second
)

// Poll tries to call the /internal/pages/status API endpoint once plus
// `retries` every `interval`.
// TODO: Remove in https://gitlab.com/gitlab-org/gitlab/-/issues/218357
func (g *Gitlab) Poll(retries int, interval time.Duration) {
	var err error
	for i := 0; i <= retries; i++ {
		log.Info("polling GitLab internal pages status API")
		err = g.checker.Status()
		if err == nil {
			log.Info("GitLab internal pages status API connected successfully")
			g.mu.Lock()
			g.isReady = true
			g.mu.Unlock()

			// return as soon as we connect to the API
			return
		}

		time.Sleep(interval)
	}

	log.WithError(err).Errorf("polling failed after %d tries every %.2fs", retries+1, interval.Seconds())
}
