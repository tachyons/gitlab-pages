package cache

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Retriever is an utility type that performs an HTTP request with backoff in
// case of errors
type Retriever struct {
	client api.Client
}

// Retrieve retrieves a lookup response from external source with timeout and
// backoff. It has its own context with timeout.
func (r *Retriever) Retrieve(domain string) (lookup *api.Lookup) {
	ctx, cancel := context.WithTimeout(context.Background(), retrievalTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Debug("retrieval context done")
		lookup = &api.Lookup{Error: errors.New("retrieval context done")}
	case lookup = <-r.resolveWithBackoff(ctx, domain):
		log.Debug("retrieval response sent")
	}

	return lookup
}

func (r *Retriever) resolveWithBackoff(ctx context.Context, domain string) <-chan *api.Lookup {
	response := make(chan *api.Lookup)

	go func() {
		var lookup *api.Lookup

		for i := 1; i <= maxRetrievalRetries; i++ {
			lookup = r.client.GetLookup(ctx, domain)

			if lookup.Error != nil {
				time.Sleep(maxRetrievalInterval)
			} else {
				break
			}
		}

		response <- lookup
		close(response)
	}()

	return response
}
