package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/namsral/flag"
	"github.com/stretchr/testify/require"
)

var pagesBinary = flag.String("gitlab-pages-binary", "./gitlab-pages", "Path to the gitlab-pages binary")

// TODO: Use TCP port 0 everywhere to avoid conflicts. The binary could output
// the actual port (and type of listener) for us to read in place of the
// hardcoded values below.
var listeners = []ListenSpec{
	{"http", "127.0.0.1", "37000"},
	{"http", "::1", "37000"},
	{"https", "127.0.0.1", "37001"},
	{"https", "::1", "37001"},
	{"proxy", "127.0.0.1", "37002"},
	{"proxy", "::1", "37002"},
}

var (
	httpListener  = listeners[0]
	httpsListener = listeners[2]
	proxyListener = listeners[4]
)

func skipUnlessEnabled(t *testing.T, conditions ...string) {
	t.Helper()

	if testing.Short() {
		t.Log("Acceptance tests disabled")
		t.SkipNow()
	}

	if _, err := os.Stat(*pagesBinary); os.IsNotExist(err) {
		t.Errorf("Couldn't find gitlab-pages binary at %s", *pagesBinary)
		t.FailNow()
	}

	for _, condition := range conditions {
		switch condition {
		case "not-inplace-chroot":
			if os.Getenv("TEST_DAEMONIZE") == "inplace" {
				t.Log("Not supported with -daemon-inplace-chroot")
				t.SkipNow()
			}
		default:
			t.Error("Unknown condition:", condition)
			t.FailNow()
		}
	}
}

func TestUnknownHostReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "invalid.invalid", "")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestUnknownProjectReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestGroupDomainReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestKnownHostReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	tests := []struct {
		name string
		host string
		path string
	}{
		{
			name: "lower case",
			host: "group.gitlab-example.com",
			path: "project/",
		},
		{
			name: "capital project",
			host: "group.gitlab-example.com",
			path: "CapitalProject/",
		},
		{
			name: "capital group",
			host: "CapitalGroup.gitlab-example.com",
			path: "project/",
		},
		{
			name: "capital group and project",
			host: "CapitalGroup.gitlab-example.com",
			path: "CapitalProject/",
		},
		{
			name: "subgroup",
			host: "group.gitlab-example.com",
			path: "subgroup/project/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, tt.host, tt.path)

				require.NoError(t, err)
				rsp.Body.Close()
				require.Equal(t, http.StatusOK, rsp.StatusCode)
			}
		})
	}
}

func TestNestedSubgroups(t *testing.T) {
	skipUnlessEnabled(t)

	maxNestedSubgroup := 21

	pagesRoot, err := ioutil.TempDir("", "pages-root")
	require.NoError(t, err)
	defer os.RemoveAll(pagesRoot)

	makeProjectIndex := func(subGroupPath string) {
		projectPath := path.Join(pagesRoot, "nested", subGroupPath, "project", "public")
		require.NoError(t, os.MkdirAll(projectPath, 0755))

		projectIndex := path.Join(projectPath, "index.html")
		require.NoError(t, ioutil.WriteFile(projectIndex, []byte("index"), 0644))
	}
	makeProjectIndex("")

	paths := []string{""}
	for i := 1; i < maxNestedSubgroup*2; i++ {
		subGroupPath := fmt.Sprintf("%ssub%d/", paths[i-1], i)
		paths = append(paths, subGroupPath)

		makeProjectIndex(subGroupPath)
	}

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-pages-root", pagesRoot)
	defer teardown()

	for nestingLevel, path := range paths {
		t.Run(fmt.Sprintf("nested level %d", nestingLevel), func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, "nested.gitlab-example.com", path+"project/")

				require.NoError(t, err)
				rsp.Body.Close()
				if nestingLevel <= maxNestedSubgroup {
					require.Equal(t, http.StatusOK, rsp.StatusCode)
				} else {
					require.Equal(t, http.StatusNotFound, rsp.StatusCode)
				}
			}
		})
	}
}
func TestCORSWhenDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-disable-cross-origin-requests")
	defer teardown()

	for _, spec := range listeners {
		for _, method := range []string{"GET", "OPTIONS"} {
			rsp := doCrossOriginRequest(t, method, method, spec.URL("project/"))

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSAllowsGET(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		for _, method := range []string{"GET", "OPTIONS"} {
			rsp := doCrossOriginRequest(t, method, method, spec.URL("project/"))

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, "*", rsp.Header.Get("Access-Control-Allow-Origin"))
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSForbidsPOST(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp := doCrossOriginRequest(t, "OPTIONS", "POST", spec.URL("project/"))

		require.Equal(t, http.StatusOK, rsp.StatusCode)
		require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
		require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
	}
}

func TestCustomHeaders(t *testing.T) {
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-header", "X-Test1:Testing1", "-header", "X-Test2:Testing2")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:", "project/")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		require.Equal(t, "Testing1", rsp.Header.Get("X-Test1"))
		require.Equal(t, "Testing2", rsp.Header.Get("X-Test2"))
	}
}

func doCrossOriginRequest(t *testing.T, method, reqMethod, url string) *http.Response {
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)

	req.Host = "group.gitlab-example.com"
	req.Header.Add("Origin", "example.com")
	req.Header.Add("Access-Control-Request-Method", reqMethod)

	var rsp *http.Response
	err = fmt.Errorf("no request was made")
	for start := time.Now(); time.Since(start) < 1*time.Second; {
		rsp, err = DoPagesRequest(t, req)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, err)

	rsp.Body.Close()
	return rsp
}

func TestKnownHostWithPortReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:"+spec.Port, "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}

}

func TestHttpToHttpsRedirectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpToHttpsRedirectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=true")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))
	require.Equal(t, "https://group.gitlab-example.com/project/", rsp.Header.Get("Location"))

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyGroupEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.https-only.gitlab-example.com", "project1/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyGroupDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.https-only.gitlab-example.com", "project2/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyProjectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "test.my-domain.com", "/index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyProjectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "test2.my-domain.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyDomainDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "no.cert.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestPrometheusMetricsCanBeScraped(t *testing.T) {
	skipUnlessEnabled(t)
	listener := []ListenSpec{{"http", "127.0.0.1", "37003"}}
	teardown := RunPagesProcess(t, *pagesBinary, listener, ":42345")
	defer teardown()

	resp, err := http.Get("http://localhost:42345/metrics")
	require.NoError(t, err)

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Contains(t, string(body), "gitlab_pages_http_in_flight_requests 0")
	require.Contains(t, string(body), "gitlab_pages_served_domains 16")
	require.Contains(t, string(body), "gitlab_pages_domains_failed_total 0")
	require.Contains(t, string(body), "gitlab_pages_domains_updated_total 1")
	require.Contains(t, string(body), "gitlab_pages_last_domain_update_seconds gauge")
	require.Contains(t, string(body), "gitlab_pages_domains_configuration_update_duration gauge")
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_hit 0")
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_miss 0")
	require.Contains(t, string(body), "gitlab_pages_serverless_requests 0")
	require.Contains(t, string(body), "gitlab_pages_serverless_latency_sum 0")
}

func TestStatusPage(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-pages-status=/@statuscheck")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestStatusNotYetReady(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", "-pages-status=/@statuscheck", "-pages-root=shared/invalid-pages")
	defer teardown()

	waitForRoundtrips(t, listeners, 5*time.Second)
	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestPageNotAvailableIfNotLoaded(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", "-pages-root=shared/invalid-pages")
	defer teardown()
	waitForRoundtrips(t, listeners, 5*time.Second)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestObscureMIMEType(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "")
	defer teardown()

	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/file.webmanifest")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusOK, rsp.StatusCode)
	mt, _, err := mime.ParseMediaType(rsp.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "application/manifest+json", mt)
}

func TestArtifactProxyRequest(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	transport := (TestHTTPSClient.Transport).(*http.Transport)
	defer func(t time.Duration) {
		transport.ResponseHeaderTimeout = t
	}(transport.ResponseHeaderTimeout)
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
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()
	defer testServer.Close()

	tests := []struct {
		name         string
		host         string
		path         string
		status       int
		binaryOption string
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
			binaryOption: "",
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
			binaryOption: "",
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
			binaryOption: "-artifacts-server-timeout=1",
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
			binaryOption: "",
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
			binaryOption: "",
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

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			teardown := RunPagesProcessWithSSLCertFile(
				t,
				*pagesBinary,
				listeners,
				"",
				certFile,
				"-artifacts-server="+artifactServerURL,
				tt.binaryOption,
			)
			defer teardown()

			resp, err := GetPageFromListener(t, httpListener, tt.host, tt.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.status, resp.StatusCode)
			require.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))

			if !((tt.status == http.StatusBadGateway) || (tt.status == http.StatusNotFound) || (tt.status == http.StatusInternalServerError)) {
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
	skipUnlessEnabled(t, "not-inplace-chroot")

	setupTransport(t)

	testServer := makeGitLabPagesAccessStub(t)

	keyFile, certFile := CreateHTTPSFixtureFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()
	defer testServer.Close()

	tests := []struct {
		name         string
		host         string
		path         string
		status       int
		binaryOption string
	}{
		{
			name:         "basic proxied request for private project",
			host:         "group.gitlab-example.com",
			path:         "/-/private/-/jobs/1/artifacts/200.html",
			status:       http.StatusOK,
			binaryOption: "",
		},
		{
			name:         "basic proxied request for subgroup",
			host:         "group.gitlab-example.com",
			path:         "/-/subgroup/private/-/jobs/1/artifacts/200.html",
			status:       http.StatusOK,
			binaryOption: "",
		},
		{
			name:         "502 error while attempting to proxy",
			host:         "group.gitlab-example.com",
			path:         "/-/private/-/jobs/1/artifacts/delayed_200.html",
			status:       http.StatusBadGateway,
			binaryOption: "-artifacts-server-timeout=1",
		},
		{
			name:         "Proxying 404 from server",
			host:         "group.gitlab-example.com",
			path:         "/-/private/-/jobs/1/artifacts/404.html",
			status:       http.StatusNotFound,
			binaryOption: "",
		},
		{
			name:         "Proxying 500 from server",
			host:         "group.gitlab-example.com",
			path:         "/-/private/-/jobs/1/artifacts/500.html",
			status:       http.StatusInternalServerError,
			binaryOption: "",
		},
	}

	// Ensure the IP address is used in the URL, as we're relying on IP SANs to
	// validate
	artifactServerURL := testServer.URL + "/api/v4"
	t.Log("Artifact server URL", artifactServerURL)

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			teardown := RunPagesProcessWithSSLCertFile(
				t,
				*pagesBinary,
				listeners,
				"",
				certFile,
				"-artifacts-server="+artifactServerURL,
				"-auth-client-id=1",
				"-auth-client-secret=1",
				"-auth-server="+testServer.URL,
				"-auth-redirect-uri=https://projects.gitlab-example.com/auth",
				"-auth-secret=something-very-secret",
				tt.binaryOption,
			)
			defer teardown()

			resp, err := GetRedirectPage(t, httpListener, tt.host, tt.path)
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

func TestEnvironmentVariablesConfig(t *testing.T) {
	skipUnlessEnabled(t)
	os.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()
	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com:", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	skipUnlessEnabled(t)
	os.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, []ListenSpec{httpsListener}, "")
	defer teardown()

	for _, listener := range []ListenSpec{httpListener, httpsListener} {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		rsp.Body.Close()

		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestMultiFlagEnvironmentVariables(t *testing.T) {
	skipUnlessEnabled(t)
	listenSpecs := []ListenSpec{{"http", "127.0.0.1", "37001"}, {"http", "127.0.0.1", "37002"}}
	envVarValue := fmt.Sprintf("%s,%s", net.JoinHostPort("127.0.0.1", "37001"), net.JoinHostPort("127.0.0.1", "37002"))

	os.Setenv("LISTEN_HTTP", envVarValue)
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcess(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()

	for _, listener := range listenSpecs {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestKnownHostInReverseProxySetupReturns200(t *testing.T) {
	skipUnlessEnabled(t)

	var listeners = []ListenSpec{
		{"proxy", "127.0.0.1", "37002"},
		{"proxy", "::1", "37002"},
	}

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetProxiedPageFromListener(t, spec, "localhost", "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestWhenAuthIsDisabledPrivateIsNotAccessible(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
}

func TestWhenAuthIsEnabledPrivateWillRedirectToAuthorize(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	rsp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))

	url, err = url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	require.Equal(t, "https", url.Scheme)
	require.Equal(t, "gitlab-auth.com", url.Host)
	require.Equal(t, "/oauth/authorize", url.Path)
	require.Equal(t, "1", url.Query().Get("client_id"))
	require.Equal(t, "https://projects.gitlab-example.com/auth", url.Query().Get("redirect_uri"))
	require.NotEqual(t, "", url.Query().Get("state"))
}

func TestWhenAuthDeniedWillCauseUnauthorized(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?error=access_denied")

	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
}
func TestWhenLoginCallbackWithWrongStateShouldFail(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	// Go to auth page with wrong state will cause failure
	authrsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?code=0&state=0")

	require.NoError(t, err)
	defer authrsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, authrsp.StatusCode)
}

func TestWhenLoginCallbackWithCorrectStateWithoutEndpoint(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetPageFromListenerWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
		url.Query().Get("state"), cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will cause 503 because token endpoint is not available
	require.Equal(t, http.StatusServiceUnavailable, authrsp.StatusCode)
}

// makeGitLabPagesAccessStub provides a stub *httptest.Server to check pages_access API call.
// the result is based on the project id.
//
// Project IDs must be 4 digit long and the following rules applies:
//   1000-1999: Ok
//   2000-2999: Unauthorized
//   3000-3999: Invalid token
func makeGitLabPagesAccessStub(t *testing.T) *httptest.Server {
	return httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			require.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			if handleAccessControlArtifactRequests(t, w, r) {
				return
			}
			handleAccessControlRequests(t, w, r)
		}
	}))
}

var existingAcmeTokenPath = "/.well-known/acme-challenge/existingtoken"
var notexistingAcmeTokenPath = "/.well-known/acme-challenge/notexistingtoken"

func TestAcmeChallengesWhenItIsConfigured(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-gitlab-server=https://gitlab-acme.com")
	defer teardown()

	t.Run("When domain folder contains requested acme challenge it responds with it", func(t *testing.T) {
		rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
			existingAcmeTokenPath)

		defer rsp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		body, _ := ioutil.ReadAll(rsp.Body)
		require.Equal(t, "this is token\n", string(body))
	})

	t.Run("When domain folder doesn't contains requested acme challenge it redirects to GitLab",
		func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				notexistingAcmeTokenPath)

			defer rsp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			require.Equal(t, url.String(), "https://gitlab-acme.com/-/acme-challenge?domain=withacmechallenge.domain.com&token=notexistingtoken")
		},
	)
}

func TestAcmeChallengesWhenItIsNotConfigured(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "")
	defer teardown()

	t.Run("When domain folder contains requested acme challenge it responds with it", func(t *testing.T) {
		rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
			existingAcmeTokenPath)

		defer rsp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		body, _ := ioutil.ReadAll(rsp.Body)
		require.Equal(t, "this is token\n", string(body))
	})

	t.Run("When domain folder doesn't contains requested acme challenge it returns 404",
		func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				notexistingAcmeTokenPath)

			defer rsp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, http.StatusNotFound, rsp.StatusCode)
		},
	)
}

func handleAccessControlArtifactRequests(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	authorization := r.Header.Get("Authorization")

	switch {
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/delayed_200.html`).MatchString(r.URL.Path):
		sleepIfAuthorized(t, authorization, w)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/404.html`).MatchString(r.URL.Path):
		w.WriteHeader(http.StatusNotFound)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/500.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusInternalServerError)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusOK)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/subgroup/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusOK)
		return true
	default:
		return false
	}
}

func handleAccessControlRequests(t *testing.T, w http.ResponseWriter, r *http.Request) {
	allowedProjects := regexp.MustCompile(`/api/v4/projects/1\d{3}/pages_access`)
	deniedProjects := regexp.MustCompile(`/api/v4/projects/2\d{3}/pages_access`)
	invalidTokenProjects := regexp.MustCompile(`/api/v4/projects/3\d{3}/pages_access`)

	switch {
	case allowedProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	case deniedProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
	case invalidTokenProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
	default:
		t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
	}
}

func returnIfAuthorized(t *testing.T, authorization string, w http.ResponseWriter, status int) {
	if authorization != "" {
		require.Equal(t, "Bearer abc", authorization)
		w.WriteHeader(status)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func sleepIfAuthorized(t *testing.T, authorization string, w http.ResponseWriter) {
	if authorization != "" {
		require.Equal(t, "Bearer abc", authorization)
		time.Sleep(2 * time.Second)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

}

func TestAccessControlUnderCustomDomain(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "private.domain.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := url.Query().Get("state")
	require.Equal(t, url.Query().Get("domain"), "http://private.domain.com")

	pagesrsp, err := GetRedirectPage(t, httpListener, url.Host, url.Path+"?"+url.RawQuery)
	require.NoError(t, err)
	defer pagesrsp.Body.Close()

	pagescookie := pagesrsp.Header.Get("Set-Cookie")

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetRedirectPageWithCookie(t, httpListener, "projects.gitlab-example.com", "/auth?code=1&state="+
		state, pagescookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "private.domain.com", url.Host)
	require.Equal(t, "1", url.Query().Get("code"))
	require.Equal(t, state, url.Query().Get("state"))

	// Run auth callback in custom domain
	authrsp, err = GetRedirectPageWithCookie(t, httpListener, "private.domain.com", "/auth?code=1&state="+
		state, cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will redirect to the page
	cookie = authrsp.Header.Get("Set-Cookie")
	require.Equal(t, http.StatusFound, authrsp.StatusCode)

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "http://private.domain.com/", url.String())

	// Fetch page in custom domain
	authrsp, err = GetRedirectPageWithCookie(t, httpListener, "private.domain.com", "/", cookie)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestAccessControlUnderCustomDomainWithHTTPSProxy(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

	rsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com", "/", "", true)
	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := url.Query().Get("state")
	require.Equal(t, url.Query().Get("domain"), "https://private.domain.com")
	pagesrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, url.Host, url.Path+"?"+url.RawQuery, "", true)
	require.NoError(t, err)
	defer pagesrsp.Body.Close()

	pagescookie := pagesrsp.Header.Get("Set-Cookie")

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener,
		"projects.gitlab-example.com", "/auth?code=1&state="+state,
		pagescookie, true)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "private.domain.com", url.Host)
	require.Equal(t, "1", url.Query().Get("code"))
	require.Equal(t, state, url.Query().Get("state"))

	// Run auth callback in custom domain
	authrsp, err = GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com",
		"/auth?code=1&state="+state, cookie, true)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will redirect to the page
	cookie = authrsp.Header.Get("Set-Cookie")
	require.Equal(t, http.StatusFound, authrsp.StatusCode)

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "https://private.domain.com/", url.String())
	// Fetch page in custom domain
	authrsp, err = GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com", "/",
		cookie, true)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestAccessControlGroupDomain404RedirectsAuth(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusFound, rsp.StatusCode)
	// Redirects to the projects under gitlab pages domain for authentication flow
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "projects.gitlab-example.com", url.Host)
	require.Equal(t, "/auth", url.Path)
}
func TestAccessControlProject404DoesNotRedirect(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/project/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func setupTransport(t *testing.T) {
	transport := (TestHTTPSClient.Transport).(*http.Transport)
	defer func(t time.Duration) {
		transport.ResponseHeaderTimeout = t
	}(transport.ResponseHeaderTimeout)
	transport.ResponseHeaderTimeout = 5 * time.Second
}

func TestAccessControl(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	setupTransport(t)

	keyFile, certFile := CreateHTTPSFixtureFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

	testServer := makeGitLabPagesAccessStub(t)
	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()
	defer testServer.Close()

	tests := []struct {
		host         string
		path         string
		status       int
		redirectBack bool
		name         string
	}{
		{
			name:         "project with access",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		{
			name:         "project without access",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		{
			name:         "invalid token test should redirect back",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		{
			name:         "no project should redirect to login and then return 404",
			host:         "group.auth.gitlab-example.com",
			path:         "/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
		{
			name:         "no project should redirect to login and then return 404",
			host:         "nonexistent.gitlab-example.com",
			path:         "/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		}, // subgroups
		{
			name:         "[subgroup] project with access",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		{
			name:         "[subgroup] project without access",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		{
			name:         "[subgroup] invalid token test should redirect back",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		{
			name:         "[subgroup] no project should redirect to login and then return 404",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
		{
			name:         "[subgroup] no project should redirect to login and then return 404",
			host:         "nonexistent.gitlab-example.com",
			path:         "/subgroup/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			teardown := RunPagesProcessWithAuthServerWithSSL(t, *pagesBinary, listeners, "", certFile, testServer.URL)
			defer teardown()

			rsp, err := GetRedirectPage(t, httpsListener, tt.host, tt.path)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusFound, rsp.StatusCode)
			cookie := rsp.Header.Get("Set-Cookie")

			// Redirects to the projects under gitlab pages domain for authentication flow
			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "projects.gitlab-example.com", url.Host)
			require.Equal(t, "/auth", url.Path)
			state := url.Query().Get("state")

			rsp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusFound, rsp.StatusCode)
			pagesDomainCookie := rsp.Header.Get("Set-Cookie")

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

			// server returns the ticket, user will be redirected to the project page
			require.Equal(t, http.StatusFound, authrsp.StatusCode)
			cookie = authrsp.Header.Get("Set-Cookie")
			rsp, err = GetRedirectPageWithCookie(t, httpsListener, tt.host, tt.path, cookie)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, tt.status, rsp.StatusCode)
			require.Equal(t, "", rsp.Header.Get("Cache-Control"))

			if tt.redirectBack {
				url, err = url.Parse(rsp.Header.Get("Location"))
				require.NoError(t, err)

				require.Equal(t, "https", url.Scheme)
				require.Equal(t, tt.host, url.Host)
				require.Equal(t, tt.path, url.Path)
			}
		})
	}
}

func TestAcceptsSupportedCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	tlsConfig := &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}
	client, cleanup := ClientWithConfig(tlsConfig)
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.NoError(t, err)
}

func tlsConfigWithInsecureCiphersOnly() *tls.Config {
	return &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		},
		MaxVersion: tls.VersionTLS12, // ciphers for TLS1.3 are not configurable and will work if enabled
	}
}

func TestRejectsUnsupportedCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.Error(t, err)
	require.Nil(t, rsp)
}

func TestEnableInsecureCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-insecure-ciphers")
	defer teardown()

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.NoError(t, err)
}

func TestTLSVersions(t *testing.T) {
	skipUnlessEnabled(t)

	tests := map[string]struct {
		tlsMin      string
		tlsMax      string
		tlsClient   uint16
		expectError bool
	}{
		"client version not supported":             {tlsMin: "tls1.1", tlsMax: "tls1.2", tlsClient: tls.VersionTLS10, expectError: true},
		"client version supported":                 {tlsMin: "tls1.1", tlsMax: "tls1.2", tlsClient: tls.VersionTLS12, expectError: false},
		"client and server using default settings": {tlsMin: "", tlsMax: "", tlsClient: 0, expectError: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := []string{}
			if tc.tlsMin != "" {
				args = append(args, "-tls-min-version", tc.tlsMin)
			}
			if tc.tlsMax != "" {
				args = append(args, "-tls-max-version", tc.tlsMax)
			}

			teardown := RunPagesProcess(t, *pagesBinary, listeners, "", args...)
			defer teardown()

			tlsConfig := &tls.Config{}
			if tc.tlsClient != 0 {
				tlsConfig.MinVersion = tc.tlsClient
				tlsConfig.MaxVersion = tc.tlsClient
			}
			client, cleanup := ClientWithConfig(tlsConfig)
			defer cleanup()

			rsp, err := client.Get(httpsListener.URL("/"))

			if rsp != nil {
				rsp.Body.Close()
			}

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGitlabDomainsSource(t *testing.T) {
	skipUnlessEnabled(t)

	source := NewGitlabDomainsSourceStub(t)
	defer source.Close()

	gitlabSourceConfig := `
domains:
  enabled:
    - new-source-test.gitlab.io
  broken: pages-broken-poc.gitlab.io
`
	gitlabSourceConfigFile, cleanupGitlabSourceConfigFile := CreateGitlabSourceConfigFixtureFile(t, gitlabSourceConfig)
	defer cleanupGitlabSourceConfigFile()

	gitlabSourceConfigFile = "GITLAB_SOURCE_CONFIG_FILE=" + gitlabSourceConfigFile

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

	pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey}

	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, listeners, "", []string{gitlabSourceConfigFile}, pagesArgs...)
	defer teardown()

	t.Run("when a domain exists", func(t *testing.T) {
		response, err := GetPageFromListener(t, httpListener, "new-source-test.gitlab.io", "/my/pages/project/")
		require.NoError(t, err)

		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)

		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "New Pages GitLab Source TEST OK\n", string(body))
	})

	t.Run("when a domain does not exists", func(t *testing.T) {
		response, err := GetPageFromListener(t, httpListener, "non-existent-domain.gitlab.io", "/path")
		defer response.Body.Close()
		require.NoError(t, err)

		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("broken domain is requested", func(t *testing.T) {
		response, err := GetPageFromListener(t, httpListener, "pages-broken-poc.gitlab.io", "index.html")
		require.NoError(t, err)

		defer response.Body.Close()

		require.Equal(t, http.StatusBadGateway, response.StatusCode)
	})
}
