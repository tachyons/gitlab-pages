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
		https  bool
		host   string
		domain *domain.D
	}{
		{
			name:   "https",
			https:  true,
			host:   "githost.io",
			domain: &domain.D{},
		},
		{
			name:   "http",
			https:  false,
			host:   "githost.io",
			domain: &domain.D{},
		},
		{
			name:   "no_domain",
			https:  false,
			host:   "githost.io",
			domain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			req = request.WithHTTPSFlag(req, tt.https)
			req = request.WithHostAndDomain(req, tt.host, tt.domain)

			got := getExtraLogFields(req)
			require.Equal(t, got["pages_https"], tt.https)
			require.Equal(t, got["pages_host"], tt.host)
			require.Equal(t, got["pages_project_id"], uint64(0x0))
		})
	}
}
