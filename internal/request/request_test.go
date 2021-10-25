package request

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsHTTPS(t *testing.T) {
	t.Run("when scheme is http", func(t *testing.T) {
		httpRequest, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		httpRequest.URL.Scheme = SchemeHTTP
		require.False(t, IsHTTPS(httpRequest))
	})

	t.Run("when scheme is https", func(t *testing.T) {
		httpsRequest, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		httpsRequest.URL.Scheme = SchemeHTTPS
		require.True(t, IsHTTPS(httpsRequest))
	})
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
