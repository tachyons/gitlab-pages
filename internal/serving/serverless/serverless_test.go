package serverless

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

func TestServeFileHTTP(t *testing.T) {
	config, err := NewClusterConfig(fixture.Certificate, fixture.Key)
	require.NoError(t, err)

	mux := http.NewServeMux()
	cluster := httptest.NewUnstartedServer(mux)

	cluster.TLS = &tls.Config{
		Certificates: []tls.Certificate{config.Certificate},
		RootCAs:      config.RootCerts,
	}

	cluster.StartTLS()
	defer cluster.Close()

	clusterURL, err := url.Parse(cluster.URL)
	require.NoError(t, err)

	serverless := New(Cluster{
		Hostname: "knative.gitlab-example.com",
		Address:  clusterURL.Hostname(),
		Port:     clusterURL.Port(),
		Config:   config,
	})

	t.Run("when proxying simple request to a cluster", func(t *testing.T) {
		writer := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "http://example.gitlab.com/simple/proxy", nil)
		request.Header.Set("X-Real-IP", "127.0.0.105")

		handler := serving.Handler{Writer: writer, Request: request}

		mux.HandleFunc("/simple/proxy", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GitLab Pages Daemon", r.Header.Get("User-Agent"))
			assert.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
			assert.Contains(t, r.Header.Get("X-Forwarded-For"), "127.0.0.105")
		})

		served := serverless.ServeFileHTTP(handler)
		result := writer.Result()

		assert.True(t, served)
		assert.Equal(t, http.StatusOK, result.StatusCode)
	})
}
