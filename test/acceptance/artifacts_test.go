package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestArtifactProxyRequest(t *testing.T) {
	transport := (TestHTTPSClient.Transport).(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second

	content := "<!DOCTYPE html><html><head><title>Title of the document</title></head><body></body></html>"
	contentLength := int64(len(content))
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RawPath {
		case "/api/v4/projects/group%2Fproject/jobs/1/artifacts/delayed_200.html":
			time.Sleep(2 * time.Second)
			fallthrough
		case "/api/v4/projects/group%2Fproject/jobs/1/artifacts/200.html",
			"/api/v4/projects/group%2Fsubgroup%2Fproject/jobs/1/artifacts/200.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, content)
		case "/api/v4/projects/group%2Fproject/jobs/1/artifacts/500.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, content)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.RawPath)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, content)
		}
	}))

	keyFile, certFile := CreateHTTPSFixtureFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()

	t.Cleanup(func() {
		os.Remove(keyFile)
		os.Remove(certFile)
		testServer.Close()
	})

	tests := []struct {
		name         string
		host         string
		path         string
		status       int
		content      string
		length       int64
		cacheControl string
		contentType  string
	}{
		{
			name:         "basic proxied request",
			host:         "group.gitlab-example.com",
			path:         "/-/project/-/jobs/1/artifacts/200.html",
			status:       http.StatusOK,
			content:      content,
			length:       contentLength,
			cacheControl: "max-age=3600",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "basic proxied request for subgroup",
			host:         "group.gitlab-example.com",
			path:         "/-/subgroup/project/-/jobs/1/artifacts/200.html",
			status:       http.StatusOK,
			content:      content,
			length:       contentLength,
			cacheControl: "max-age=3600",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "502 error while attempting to proxy",
			host:         "group.gitlab-example.com",
			path:         "/-/project/-/jobs/1/artifacts/delayed_200.html",
			status:       http.StatusBadGateway,
			content:      "",
			length:       0,
			cacheControl: "",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "Proxying 404 from server",
			host:         "group.gitlab-example.com",
			path:         "/-/project/-/jobs/1/artifacts/404.html",
			status:       http.StatusNotFound,
			content:      "",
			length:       0,
			cacheControl: "",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "Proxying 500 from server",
			host:         "group.gitlab-example.com",
			path:         "/-/project/-/jobs/1/artifacts/500.html",
			status:       http.StatusInternalServerError,
			content:      "",
			length:       0,
			cacheControl: "",
			contentType:  "text/html; charset=utf-8",
		},
	}

	// Ensure the IP address is used in the URL, as we're relying on IP SANs to
	// validate
	artifactServerURL := testServer.URL + "/api/v4"
	t.Log("Artifact server URL", artifactServerURL)

	args := []string{"-artifacts-server=" + artifactServerURL, "-artifacts-server-timeout=1"}

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withArguments(args),
		withEnv([]string{"SSL_CERT_FILE=" + certFile}),
	)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := GetPageFromListener(t, httpListener, tt.host, tt.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			require.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))

			if tt.status == http.StatusOK {
				body, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Equal(t, tt.content, string(body))
				require.Equal(t, tt.length, resp.ContentLength)
				require.Equal(t, tt.cacheControl, resp.Header.Get("Cache-Control"))
			}
		})
	}
}

func TestPrivateArtifactProxyRequest(t *testing.T) {
	setupTransport(t)

	testServer := makeGitLabPagesAccessStub(t)

	keyFile, certFile := CreateHTTPSFixtureFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()

	t.Cleanup(func() {
		os.Remove(keyFile)
		os.Remove(certFile)
		testServer.Close()
	})

	tests := []struct {
		name   string
		host   string
		path   string
		status int
	}{
		{
			name:   "basic proxied request for private project",
			host:   "group.gitlab-example.com",
			path:   "/-/private/-/jobs/1/artifacts/200.html",
			status: http.StatusOK,
		},
		{
			name:   "basic proxied request for subgroup",
			host:   "group.gitlab-example.com",
			path:   "/-/subgroup/private/-/jobs/1/artifacts/200.html",
			status: http.StatusOK,
		},
		{
			name:   "502 error while attempting to proxy",
			host:   "group.gitlab-example.com",
			path:   "/-/private/-/jobs/1/artifacts/delayed_200.html",
			status: http.StatusBadGateway,
		},
		{
			name:   "Proxying 404 from server",
			host:   "group.gitlab-example.com",
			path:   "/-/private/-/jobs/1/artifacts/404.html",
			status: http.StatusNotFound,
		},
		{
			name:   "Proxying 500 from server",
			host:   "group.gitlab-example.com",
			path:   "/-/private/-/jobs/1/artifacts/500.html",
			status: http.StatusInternalServerError,
		},
	}

	// Ensure the IP address is used in the URL, as we're relying on IP SANs to
	// validate
	artifactServerURL := testServer.URL + "/api/v4"
	t.Log("Artifact server URL", artifactServerURL)

	configFile := defaultConfigFileWith(t,
		"gitlab-server="+testServer.URL,
		"artifacts-server="+artifactServerURL,
		"auth-redirect-uri=https://projects.gitlab-example.com/auth",
		"artifacts-server-timeout=1")

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withArguments([]string{
			"-config=" + configFile,
		}),
		withEnv([]string{"SSL_CERT_FILE=" + certFile}),
	)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, err := GetRedirectPage(t, httpsListener, tt.host, tt.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusFound, resp.StatusCode)

			cookie := resp.Header.Get("Set-Cookie")

			// Redirects to the projects under gitlab pages domain for authentication flow
			url, err := url.Parse(resp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "projects.gitlab-example.com", url.Host)
			require.Equal(t, "/auth", url.Path)
			state := url.Query().Get("state")

			resp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)

			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusFound, resp.StatusCode)
			pagesDomainCookie := resp.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagesDomainCookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// Will redirect auth callback to correct host
			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tt.host, url.Host)
			require.Equal(t, "/auth", url.Path)

			// Request auth callback in project domain
			authrsp, err = GetRedirectPageWithCookie(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery, cookie)
			require.NoError(t, err)
			defer authrsp.Body.Close()

			// server returns the ticket, user will be redirected to the project page
			require.Equal(t, http.StatusFound, authrsp.StatusCode)
			cookie = authrsp.Header.Get("Set-Cookie")
			resp, err = GetRedirectPageWithCookie(t, httpsListener, tt.host, tt.path, cookie)

			require.Equal(t, tt.status, resp.StatusCode)

			require.NoError(t, err)
			defer resp.Body.Close()
		})
	}
}
