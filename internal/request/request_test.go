package request

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

func TestGetDomainPanic(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	assert.Panics(t, func() {
		GetDomain(r)
	})
}

func TestWithDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain *domain.D
	}{
		{
			name:   "values",
			domain: &domain.D{},
		},
		{
			name:   "no_host",
			domain: &domain.D{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			r = WithDomain(r, tt.domain)
			assert.Exactly(t, tt.domain, GetDomain(r))
		})
	}
}
