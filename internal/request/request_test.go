package request

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

func TestWithHTTPSFlag(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	httpsRequest := WithHTTPSFlag(r, true)
	assert.True(t, IsHTTPS(httpsRequest))

	httpRequest := WithHTTPSFlag(r, false)
	assert.False(t, IsHTTPS(httpRequest))
}

func TestPanics(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	assert.Panics(t, func() {
		IsHTTPS(r)
	})

	assert.Panics(t, func() {
		GetHost(r)
	})

	assert.Panics(t, func() {
		GetDomain(r)
	})
}

func TestWithHostAndDomain(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		domain *domain.D
	}{
		{
			name:   "values",
			host:   "gitlab.com",
			domain: &domain.D{},
		},
		{
			name:   "no_host",
			host:   "",
			domain: &domain.D{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			r = WithHostAndDomain(r, tt.host, tt.domain)
			assert.Exactly(t, tt.domain, GetDomain(r))
			assert.Equal(t, tt.host, GetHost(r))
		})
	}
}
