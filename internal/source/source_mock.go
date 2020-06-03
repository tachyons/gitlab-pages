package source

import (
	"github.com/stretchr/testify/mock"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// MockSource can be used for testing
type MockSource struct {
	mock.Mock
}

// GetDomain is a mocked function
func (m *MockSource) GetDomain(name string) (*domain.Domain, error) {
	args := m.Called(name)

	return args.Get(0).(*domain.Domain), args.Error(1)
}

// NewMockSource returns a new Source mock for testing
func NewMockSource() *MockSource {
	return &MockSource{}
}