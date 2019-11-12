package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var maxRetrievalInterval = time.Second

// Retriever is an utility type that performs an HTTP request with backoff in
// case of errors
type Retriever struct {
	client  Resolver
	ctx     context.Context
	timeout time.Duration
}

// Retrieve schedules a retrieval of a response and return a channel that the
// response is going to be sent to
func (r *Retriever) Retrieve(domain string) <-chan Lookup {
	response := make(chan Lookup)

	go r.retrieveWithTimeout(domain, response)

	return response
}

func (r *Retriever) retrieveWithTimeout(domain string, response chan<- Lookup) {
	newctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	defer cancel()

	var lookup Lookup

	select {
	case <-newctx.Done():
		response <- Lookup{Status: 502, Error: errors.New("context timeout")}
		fmt.Println("retrieval context done") // TODO logme
	case lookup = <-r.resolveWithBackoff(newctx, domain):
		response <- lookup
		fmt.Println("retrieval response sent") // TODO logme
	}

	close(response)
}

func (r *Retriever) resolveWithBackoff(ctx context.Context, domain string) <-chan Lookup {
	response := make(chan Lookup)

	go func(response chan<- Lookup) {
		var lookup Lookup

		for i := 1; i <= 3; i++ {
			lookup = r.client.Resolve(ctx, domain)

			if lookup.Error != nil {
				time.Sleep(maxRetrievalInterval)
			} else {
				break
			}
		}

		response <- lookup
		close(response)
	}(response)

	return response
}
