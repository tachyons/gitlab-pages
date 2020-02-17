package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

func TestWithHTTPSFlag(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	httpsRequest := WithHTTPSFlag(r, true)
	httpsRequest.URL.Scheme = SchemeHTTPS
	require.True(t, IsHTTPS(httpsRequest))

	httpRequest := WithHTTPSFlag(r, false)
	httpsRequest.URL.Scheme = SchemeHTTP
	require.False(t, IsHTTPS(httpRequest))

}

func TestIsHTTPS(t *testing.T) {
	hook := test.NewGlobal()

	tests := []struct {
		name   string
		flag   bool
		scheme string
	}{
		{
			name:   "https",
			flag:   true,
			scheme: "https",
		},
		{
			name:   "http",
			flag:   false,
			scheme: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)
			r.URL.Scheme = tt.scheme

			httpsRequest := WithHTTPSFlag(r, tt.flag)

			got := IsHTTPS(httpsRequest)
			require.Equal(t, tt.flag, got)
		})
	}

}

func TestPanics(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	require.Panics(t, func() {
		GetHost(r)
	})

	require.Panics(t, func() {
		GetDomain(r)
	})
}

func TestWithHostAndDomain(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		domain *domain.Domain
	}{
		{
			name:   "values",
			host:   "gitlab.com",
			domain: &domain.Domain{},
		},
		{
			name:   "no_host",
			host:   "",
			domain: &domain.Domain{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			r = WithHostAndDomain(r, tt.host, tt.domain)
			require.Exactly(t, tt.domain, GetDomain(r))
			require.Equal(t, tt.host, GetHost(r))
		})
	}
}

func TestGetHostWithoutPort(t *testing.T) {
	t.Run("when port component is provided", func(t *testing.T) {
		request := httptest.NewRequest("GET", "https://example.com:443", nil)
		request.Host = "my.example.com:8080"

		host := GetHostWithoutPort(request)

		require.Equal(t, "my.example.com", host)
	})

	t.Run("when port component is not provided", func(t *testing.T) {
		request := httptest.NewRequest("GET", "http://example.com", nil)
		request.Host = "my.example.com"

		host := GetHostWithoutPort(request)

		require.Equal(t, "my.example.com", host)
	})
}
