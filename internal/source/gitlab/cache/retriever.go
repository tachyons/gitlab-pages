package cache

import (
	"context"
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

	select {
	case <-newctx.Done():
		fmt.Println("retrieval context done") // TODO logme
	// TODO this waits for the response channel read instead of the resolution
	case response <- r.resolveWithBackoff(newctx, domain):
		fmt.Println("retrieval response sent") // TODO logme
	}

	close(response)
}

func (r *Retriever) resolveWithBackoff(ctx context.Context, domain string) (lookup Lookup) {
	// TODO do we want to create yet another goroutine for this to make it
	// possible to wait for a timeout in a more efficient way in the calling
	// select?
	for i := 1; i <= 3; i++ {
		lookup = r.client.Resolve(ctx, domain)

		if lookup.Err != nil {
			time.Sleep(maxRetrievalInterval)
		} else {
			break
		}
	}

	return lookup
}
