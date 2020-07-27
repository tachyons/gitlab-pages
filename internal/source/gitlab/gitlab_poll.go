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
// TODO: Remove in https://gitlab.com/gitlab-org/gitlab/-/issues/218357
func (g *Gitlab) poll(interval, maxElapsedTime time.Duration) {
	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = interval
	backOff.MaxElapsedTime = maxElapsedTime

	op := func() error {
		log.Info("Checking GitLab internal API availability")

		return g.client.Status()
	}

	err := backoff.Retry(op, backOff)
	if err != nil {
		// Handle error.
		log.WithError(err).Errorf("Failed to connect to the internal GitLab API after %.2fs", interval.Seconds())
		return
	}

	g.mu.Lock()
	g.isReady = true
	g.mu.Unlock()
	log.Info("GitLab internal pages status API connected successfully")
	//
	// var err error
	// for i := 0; i <= retries; i++ {
	// 	log.Info("polling GitLab internal pages status API")
	// 	err = g.checker.Status()
	// 	if err == nil {
	// 		log.Info("GitLab internal pages status API connected successfully")
	// 		g.mu.Lock()
	// 		g.isReady = true
	// 		g.mu.Unlock()
	//
	// 		// return as soon as we connect to the API
	// 		return
	// 	}
	//
	// 	time.Sleep(interval)
	// }
	//
	// log.WithError(err).Errorf("polling failed after %d tries every %.2fs", retries+1, interval.Seconds())
}
