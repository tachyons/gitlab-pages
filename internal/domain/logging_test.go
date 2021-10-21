package domain

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

type resolver struct {
	err error
	f   func(*http.Request) *serving.LookupPath
}

func (r *resolver) Resolve(req *http.Request) (*serving.Request, error) {
	if r.f != nil {
		return &serving.Request{LookupPath: r.f(req)}, nil
	}

	return nil, r.err
}

func TestDomainLogFields(t *testing.T) {
	domainWithResolver := New("", "", "", &resolver{f: func(*http.Request) *serving.LookupPath {
		return &serving.LookupPath{
			ServingType: "file",
			ProjectID:   100,
			Prefix:      "/prefix",
		}
	}})

	tests := map[string]struct {
		domain         *Domain
		host           string
		expectedFields log.Fields
	}{
		"nil_domain_returns_empty_fields": {
			domain:         nil,
			host:           "gitlab.io",
			expectedFields: log.Fields{},
		},
		"unresolved_domain_returns_error": {
			domain:         New("githost.io", "", "", &resolver{err: ErrDomainDoesNotExist}),
			host:           "gitlab.io",
			expectedFields: log.Fields{"error": ErrDomainDoesNotExist.Error()},
		},
		"domain_with_fields": {
			domain: domainWithResolver,
			host:   "gitlab.io",
			expectedFields: log.Fields{
				"pages_project_id":           uint64(100),
				"pages_project_prefix":       "/prefix",
				"pages_project_serving_type": "file",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			require.Equal(t, tt.expectedFields, tt.domain.LogFields(r))
		})
	}
}
