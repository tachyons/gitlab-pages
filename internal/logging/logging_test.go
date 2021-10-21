package logging

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
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

func TestGetExtraLogFields(t *testing.T) {
	domainWithResolver := domain.New("", "", "", &resolver{f: func(*http.Request) *serving.LookupPath {
		return &serving.LookupPath{
			ServingType: "file",
			ProjectID:   100,
			Prefix:      "/prefix",
		}
	}})

	tests := []struct {
		name                  string
		scheme                string
		host                  string
		expectedHTTPS         interface{}
		expectedHost          interface{}
		expectedProjectID     interface{}
		expectedProjectPrefix interface{}
		expectedServingType   interface{}
		expectedErrMsg        interface{}
	}{
		{
			name:                  "https",
			scheme:                request.SchemeHTTPS,
			host:                  "githost.io",
			expectedHTTPS:         true,
			expectedHost:          "githost.io",
			expectedProjectID:     uint64(100),
			expectedProjectPrefix: "/prefix",
			expectedServingType:   "file",
		},
		{
			name:                  "http",
			scheme:                request.SchemeHTTP,
			host:                  "githost.io",
			expectedHTTPS:         false,
			expectedHost:          "githost.io",
			expectedProjectID:     uint64(100),
			expectedProjectPrefix: "/prefix",
			expectedServingType:   "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			req.URL.Scheme = tt.scheme
			req = request.WithHostAndDomain(req, tt.host, domainWithResolver)

			got := getExtraLogFields(req)
			require.Equal(t, tt.expectedHTTPS, got["pages_https"])
			require.Equal(t, tt.expectedHost, got["pages_host"])
			require.Equal(t, tt.expectedProjectID, got["pages_project_id"])
			require.Equal(t, tt.expectedProjectPrefix, got["pages_project_prefix"])
			require.Equal(t, tt.expectedServingType, got["pages_project_serving_type"])
			require.Equal(t, tt.expectedErrMsg, got["error"])
		})
	}
}
