package client

import (
	"encoding/json"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// StubClient is a stubbed client used for testing
type StubClient struct {
	File string
}

// GetVirtualDomain reads a test fixture and unmarshalls it
func (c StubClient) GetVirtualDomain(host string) (*api.VirtualDomain, error) {
	f, err := os.Open(c.File)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	var domain api.VirtualDomain
	err = json.NewDecoder(f).Decode(&domain)

	return &domain, err
}
