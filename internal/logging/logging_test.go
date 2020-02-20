package logging

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

func TestGetExtraLogFields(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		domain *domain.Domain
	}{
		{
			name:   "https",
			scheme: request.SchemeHTTPS,
			host:   "githost.io",
			domain: &domain.Domain{},
		},
		{
			name:   "http",
			scheme: request.SchemeHTTP,
			host:   "githost.io",
			domain: &domain.Domain{},
		},
		{
			name:   "no_domain",
			scheme: request.SchemeHTTP,
			host:   "githost.io",
			domain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			req.URL.Scheme = tt.scheme
			req = request.WithHostAndDomain(req, tt.host, tt.domain)

			got := getExtraLogFields(req)
			require.Equal(t, got["pages_https"], tt.scheme == request.SchemeHTTPS)
			require.Equal(t, got["pages_host"], tt.host)
			require.Equal(t, got["pages_project_id"], uint64(0x0))
		})
	}
}
