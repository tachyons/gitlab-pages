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

	cluster := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GitLab Pages Daemon", r.Header.Get("User-Agent"))
	}))

	cluster.TLS = &tls.Config{
		Certificates: []tls.Certificate{config.Certificate},
		RootCAs:      config.RootCerts,
	}

	cluster.StartTLS()
	defer cluster.Close()

	clusterURL, err := url.Parse(cluster.URL)
	require.NoError(t, err)

	serverless := New(Cluster{
		Address:  clusterURL.Hostname(),
		Hostname: "knative.gitlab-example.com",
		Port:     clusterURL.Port(),
		Config:   config,
	})

	t.Run("when proxying simple request to a cluster", func(t *testing.T) {
		writer := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "http://example.gitlab.com", nil)
		handler := serving.Handler{Writer: writer, Request: request}

		assert.True(t, serverless.ServeFileHTTP(handler))
	})
}
