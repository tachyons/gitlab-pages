package client

import (
	"encoding/json"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/domain"
)

// StubClient is a stubbed client used for testing
type StubClient struct {
	file string
}

// GetVirtualDomain reads a test fixture and unmarshalls it
func (m *StubClient) GetVirtualDomain(host string) (domain *domain.VirtualDomain, err error) {
	f, err := os.Open(m.file)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(f).Decode(&domain)

	return domain, err
}

// NewStubClient return a stubbed client
func NewStubClient(fixture string) *StubClient {
	return &StubClient{file: fixture}
}
