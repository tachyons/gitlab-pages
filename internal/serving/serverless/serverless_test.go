package serverless

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

func withTestCluster(t *testing.T, cert, key string, block func(*http.ServeMux, *url.URL, *Certs)) {
	mux := http.NewServeMux()
	cluster := httptest.NewUnstartedServer(mux)

	certs, err := NewClusterCerts(fixture.Certificate, fixture.Key)
	require.NoError(t, err)

	cluster.TLS = &tls.Config{
		Certificates: []tls.Certificate{certs.Certificate},
		RootCAs:      certs.RootCerts,
	}

	cluster.StartTLS()
	defer cluster.Close()

	address, err := url.Parse(cluster.URL)
	require.NoError(t, err)

	block(mux, address, certs)
}

func TestServeFileHTTP(t *testing.T) {
	t.Run("when proxying simple request to a cluster", func(t *testing.T) {
		withTestCluster(t, fixture.Certificate, fixture.Key, func(mux *http.ServeMux, server *url.URL, certs *Certs) {
			serverless := New(
				Function{
					Name:       "my-func",
					Namespace:  "my-namespace-123",
					BaseDomain: "knative.example.com",
				},
				Cluster{
					Name:    "knative.gitlab-example.com",
					Address: server.Hostname(),
					Port:    server.Port(),
					Certs:   certs,
				},
			)

			writer := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "http://example.gitlab.com/", nil)
			handler := serving.Handler{Writer: writer, Request: request}
			request.Header.Set("X-Real-IP", "127.0.0.105")

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "my-func.my-namespace-123.knative.example.com", r.Host)
				require.Equal(t, "GitLab Pages Daemon", r.Header.Get("User-Agent"))
				require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
				require.Contains(t, r.Header.Get("X-Forwarded-For"), "127.0.0.105")
			})

			served := serverless.ServeFileHTTP(handler)
			result := writer.Result()

			require.True(t, served)
			require.Equal(t, http.StatusOK, result.StatusCode)
		})
	})

	t.Run("when proxying request with invalid hostname", func(t *testing.T) {
		withTestCluster(t, fixture.Certificate, fixture.Key, func(mux *http.ServeMux, server *url.URL, certs *Certs) {
			serverless := New(
				Function{
					Name:       "my-func",
					Namespace:  "my-namespace-123",
					BaseDomain: "knative.example.com",
				},
				Cluster{
					Name:    "knative.invalid-gitlab-example.com",
					Address: server.Hostname(),
					Port:    server.Port(),
					Certs:   certs,
				},
			)

			writer := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "http://example.gitlab.com/", nil)
			handler := serving.Handler{Writer: writer, Request: request}

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			})

			served := serverless.ServeFileHTTP(handler)
			result := writer.Result()
			body, err := ioutil.ReadAll(writer.Body)
			require.NoError(t, err)

			require.True(t, served)
			require.Equal(t, http.StatusInternalServerError, result.StatusCode)
			require.Contains(t, string(body), "cluster error: x509: certificate")
		})
	})

	t.Run("when a cluster responds with an error", func(t *testing.T) {
		withTestCluster(t, fixture.Certificate, fixture.Key, func(mux *http.ServeMux, server *url.URL, certs *Certs) {
			serverless := New(
				Function{
					Name:       "my-func",
					Namespace:  "my-namespace-123",
					BaseDomain: "knative.example.com",
				},
				Cluster{
					Name:    "knative.gitlab-example.com",
					Address: server.Hostname(),
					Port:    server.Port(),
					Certs:   certs,
				},
			)

			writer := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "http://example.gitlab.com/", nil)
			handler := serving.Handler{Writer: writer, Request: request}

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("sorry, service unavailable"))
			})

			served := serverless.ServeFileHTTP(handler)
			result := writer.Result()
			body, err := ioutil.ReadAll(writer.Body)
			require.NoError(t, err)

			require.True(t, served)
			require.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
			require.Contains(t, string(body), "sorry, service unavailable")
		})
	})

	t.Run("when a cluster responds correctly", func(t *testing.T) {
		withTestCluster(t, fixture.Certificate, fixture.Key, func(mux *http.ServeMux, server *url.URL, certs *Certs) {
			serverless := New(
				Function{
					Name:       "my-func",
					Namespace:  "my-namespace-123",
					BaseDomain: "knative.example.com",
				},
				Cluster{
					Name:    "knative.gitlab-example.com",
					Address: server.Hostname(),
					Port:    server.Port(),
					Certs:   certs,
				},
			)

			writer := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "http://example.gitlab.com/", nil)
			handler := serving.Handler{Writer: writer, Request: request}

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			served := serverless.ServeFileHTTP(handler)
			result := writer.Result()
			body, err := ioutil.ReadAll(writer.Body)
			require.NoError(t, err)

			require.True(t, served)
			require.Equal(t, http.StatusOK, result.StatusCode)
			require.Contains(t, string(body), "OK")
		})
	})
}
