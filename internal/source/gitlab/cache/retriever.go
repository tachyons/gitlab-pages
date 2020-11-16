package cache

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Retriever is an utility type that performs an HTTP request with backoff in
// case of errors
type Retriever struct {
	client               api.Client
	retrievalTimeout     time.Duration
	maxRetrievalInterval time.Duration
	maxRetrievalRetries  int
}

// NewRetriever creates a Retriever with a client
func NewRetriever(client api.Client, retrievalTimeout, maxRetrievalInterval time.Duration, maxRetrievalRetries int) *Retriever {
	return &Retriever{
		client:               client,
		retrievalTimeout:     retrievalTimeout,
		maxRetrievalInterval: maxRetrievalInterval,
		maxRetrievalRetries:  maxRetrievalRetries,
	}
}

// Retrieve retrieves a lookup response from external source with timeout and
// backoff. It has its own context with timeout.
func (r *Retriever) Retrieve(domain string) (lookup api.Lookup) {
	ctx, cancel := context.WithTimeout(context.Background(), r.retrievalTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Debug("retrieval context done")
		lookup = api.Lookup{Error: errors.New("retrieval context done")}
	case lookup = <-r.resolveWithBackoff(ctx, domain):
		log.Debug("retrieval response sent")
	}

	return lookup
}

func (r *Retriever) resolveWithBackoff(ctx context.Context, domain string) <-chan api.Lookup {
	response := make(chan api.Lookup)

	go func() {
		var lookup api.Lookup

		for i := 1; i <= r.maxRetrievalRetries; i++ {
			lookup = r.client.GetLookup(ctx, domain)

			if lookup.Error != nil && !errors.Is(lookup.Error, client.ErrDomainDoesNotExist) {
				time.Sleep(r.maxRetrievalInterval)
			} else {
				break
			}
		}

		response <- lookup
		close(response)
	}()

	return response
}
