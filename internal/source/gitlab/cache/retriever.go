package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Retriever is an utility type that performs an HTTP request with backoff in
// case of errors
type Retriever struct {
	client api.Client
}

// Retrieve retrieves a lookup response from external source with timeout and
// backoff. It has its own context with timeout.
func (r *Retriever) Retrieve(domain string) api.Lookup {
	ctx, cancel := context.WithTimeout(context.Background(), retrievalTimeout)
	defer cancel()

	var lookup api.Lookup

	select {
	case <-ctx.Done():
		fmt.Println("retrieval context done") // TODO logme
		lookup = api.Lookup{Error: errors.New("retrieval context done")}
	case lookup = <-r.resolveWithBackoff(ctx, domain):
		fmt.Println("retrieval response sent") // TODO logme
	}

	return lookup
}

func (r *Retriever) resolveWithBackoff(ctx context.Context, domain string) <-chan api.Lookup {
	response := make(chan api.Lookup)

	go func() {
		var lookup api.Lookup

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
