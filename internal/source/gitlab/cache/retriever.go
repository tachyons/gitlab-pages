package cache

import (
	"context"
	"fmt"
	"time"
)

// Retriever is an utility type that performs an HTTP request with backoff in
// case of errors
type Retriever struct {
	client  Resolver
	ctx     context.Context
	timeout time.Duration
}

// Retrieve schedules a retrieval of a response and return a channel that the
// response is going to be sent to
func (r *Retriever) Retrieve(domain string) <-chan Response {
	response := make(chan Response)

	go r.retrieveWithBackoff(domain, response)

	return response
}

func (r *Retriever) retrieveWithBackoff(domain string, response chan<- Response) {
	newctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	defer cancel()

	for i := 1; i <= 3; i++ {
		lookup, status, err := r.client.Resolve(newctx, domain)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		select {
		case <-newctx.Done():
			fmt.Println("retrieval context done") // TODO logme
		case response <- Response{lookup: lookup, status: status, err: err}:
			fmt.Println("retrieval response sent") // TODO logme
		}

		break
	}

	close(response)
}
