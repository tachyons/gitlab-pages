package client

import (
	"context"
	"encoding/json"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// StubClient is a stubbed client used for testing
type StubClient struct {
	File string
}

// Resolve implements api.Resolver
func (c StubClient) Resolve(ctx context.Context, host string) *api.Lookup {
	lookup := c.GetLookup(ctx, host)

	return &lookup
}

// GetLookup reads a test fixture and unmarshalls it
func (c StubClient) GetLookup(ctx context.Context, host string) api.Lookup {
	lookup := api.Lookup{Name: host}

	f, err := os.Open(c.File)
	if err != nil {
		lookup.Error = err
		return lookup
	}
	defer f.Close()

	lookup.Error = json.NewDecoder(f).Decode(&lookup.Domain)

	return lookup
}
