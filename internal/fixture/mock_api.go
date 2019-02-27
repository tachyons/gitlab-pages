package fixture

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

// MockAPI provides a preconfigured set of domains
// for testing purposes
type MockAPI struct{}

// RequestDomain request a host from preconfigured list of domains
func (a *MockAPI) RequestDomain(host string) (*client.DomainResponse, error) {
	if response, ok := internalConfigs[host]; ok {
		return &response, nil
	}

	return nil, errors.New("not found")
}

func (a *MockAPI) IsReady() bool {
	return true
}
