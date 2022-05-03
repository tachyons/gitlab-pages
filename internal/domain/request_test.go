package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPanics(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	require.Panics(t, func() {
		FromRequest(r)
	})
}

func TestWithHostAndDomain(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		domain *Domain
	}{
		{
			name:   "values",
			host:   "gitlab.com",
			domain: &Domain{},
		},
		{
			name:   "no_host",
			host:   "",
			domain: &Domain{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			r = ReqWithDomain(r, tt.domain)
			require.Exactly(t, tt.domain, FromRequest(r))
		})
	}
}
