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
	"testing"
	"time"

	"github.com/namsral/flag"
	"github.com/stretchr/testify/assert"
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
)

func skipUnlessEnabled(t *testing.T, conditions ...string) {
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
		assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestUnknownProjectReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestGroupDomainReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestKnownHostReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestCORSWhenDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-disable-cross-origin-requests")
	defer teardown()

	for _, spec := range listeners {
		for _, method := range []string{"GET", "OPTIONS"} {
			rsp := doCrossOriginRequest(t, method, method, spec.URL("project/"))

			assert.Equal(t, http.StatusOK, rsp.StatusCode)
			assert.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
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

			assert.Equal(t, http.StatusOK, rsp.StatusCode)
			assert.Equal(t, "*", rsp.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSForbidsPOST(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp := doCrossOriginRequest(t, "OPTIONS", "POST", spec.URL("project/"))

		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
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
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}

}

func TestHttpToHttpsRedirectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpToHttpsRedirectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=true")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)
	assert.Equal(t, 1, len(rsp.Header["Location"]))
	assert.Equal(t, "https://group.gitlab-example.com/project/", rsp.Header.Get("Location"))

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyGroupEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.https-only.gitlab-example.com", "project1/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyGroupDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.https-only.gitlab-example.com", "project2/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyProjectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "test.my-domain.com", "/index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyProjectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "test2.my-domain.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyDomainDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "no.cert.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestPrometheusMetricsCanBeScraped(t *testing.T) {
	skipUnlessEnabled(t)
	listener := []ListenSpec{{"http", "127.0.0.1", "37003"}}
	teardown := RunPagesProcess(t, *pagesBinary, listener, ":42345")
	defer teardown()

	resp, err := http.Get("http://localhost:42345/metrics")

	if assert.NoError(t, err) {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		assert.Contains(t, string(body), "gitlab_pages_http_sessions_active 0")
		assert.Contains(t, string(body), "gitlab_pages_domains_served_total 11")
	}
}

func TestStatusPage(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-pages-status=/@statuscheck")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestStatusNotYetReady(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", "-pages-status=/@statuscheck", "-pages-root=shared/invalid-pages")
	defer teardown()

	waitForRoundtrips(t, listeners, 5*time.Second)
	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestPageNotAvailableIfNotLoaded(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", "-pages-root=shared/invalid-pages")
	defer teardown()
	waitForRoundtrips(t, listeners, 5*time.Second)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
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
	assert.Equal(t, "application/manifest+json", mt)
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

	cases := []struct {
		Host         string
		Path         string
		Status       int
		BinaryOption string
		Content      string
		Length       int64
		CacheControl string
		ContentType  string
		Description  string
	}{
		{
			"group.gitlab-example.com",
			"/-/project/-/jobs/1/artifacts/200.html",
			http.StatusOK,
			"",
			content,
			contentLength,
			"max-age=3600",
			"text/html; charset=utf-8",
			"basic proxied request",
		},
		{
			"group.gitlab-example.com",
			"/-/subgroup/project/-/jobs/1/artifacts/200.html",
			http.StatusOK,
			"",
			content,
			contentLength,
			"max-age=3600",
			"text/html; charset=utf-8",
			"basic proxied request for subgroup",
		},
		{
			"group.gitlab-example.com",
			"/-/project/-/jobs/1/artifacts/delayed_200.html",
			http.StatusBadGateway,
			"-artifacts-server-timeout=1",
			"",
			0,
			"",
			"text/html; charset=utf-8",
			"502 error while attempting to proxy",
		},
		{
			"group.gitlab-example.com",
			"/-/project/-/jobs/1/artifacts/404.html",
			http.StatusNotFound,
			"",
			"",
			0,
			"",
			"text/html; charset=utf-8",
			"Proxying 404 from server",
		},
		{
			"group.gitlab-example.com",
			"/-/project/-/jobs/1/artifacts/500.html",
			http.StatusInternalServerError,
			"",
			"",
			0,
			"",
			"text/html; charset=utf-8",
			"Proxying 500 from server",
		},
	}

	// Ensure the IP address is used in the URL, as we're relying on IP SANs to
	// validate
	artifactServerURL := testServer.URL + "/api/v4"
	t.Log("Artifact server URL", artifactServerURL)

	for _, c := range cases {

		t.Run(fmt.Sprintf("Proxy Request Test: %s", c.Description), func(t *testing.T) {
			teardown := RunPagesProcessWithSSLCertFile(
				t,
				*pagesBinary,
				listeners,
				"",
				certFile,
				"-artifacts-server="+artifactServerURL,
				c.BinaryOption,
			)
			defer teardown()

			resp, err := GetPageFromListener(t, httpListener, c.Host, c.Path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, c.Status, resp.StatusCode)
			assert.Equal(t, c.ContentType, resp.Header.Get("Content-Type"))

			if !((c.Status == http.StatusBadGateway) || (c.Status == http.StatusNotFound) || (c.Status == http.StatusInternalServerError)) {
				body, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, c.Content, string(body))
				assert.Equal(t, c.Length, resp.ContentLength)
				assert.Equal(t, c.CacheControl, resp.Header.Get("Cache-Control"))
			}
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
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
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

		assert.Equal(t, http.StatusOK, rsp.StatusCode)
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
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
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
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestWhenAuthIsDisabledPrivateIsNotAccessible(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	rsp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
}

func TestWhenAuthIsEnabledPrivateWillRedirectToAuthorize(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	assert.Equal(t, http.StatusFound, rsp.StatusCode)
	assert.Equal(t, 1, len(rsp.Header["Location"]))

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	assert.Equal(t, "https", url.Scheme)
	assert.Equal(t, "gitlab-auth.com", url.Host)
	assert.Equal(t, "/oauth/authorize", url.Path)
	assert.Equal(t, "1", url.Query().Get("client_id"))
	assert.Equal(t, "https://gitlab-example.com/auth", url.Query().Get("redirect_uri"))
	assert.NotEqual(t, "", url.Query().Get("state"))
}

func TestWhenAuthDeniedWillCauseUnauthorized(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpsListener, "gitlab-example.com", "/auth?error=access_denied")

	require.NoError(t, err)
	defer rsp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
}
func TestWhenLoginCallbackWithWrongStateShouldFail(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	// Go to auth page with wrong state will cause failure
	authrsp, err := GetPageFromListener(t, httpsListener, "gitlab-example.com", "/auth?code=0&state=0")

	require.NoError(t, err)
	defer authrsp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, authrsp.StatusCode)
}

func TestWhenLoginCallbackWithCorrectStateWithoutEndpoint(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetPageFromListenerWithCookie(t, httpsListener, "gitlab-example.com", "/auth?code=1&state="+
		url.Query().Get("state"), cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will cause 503 because token endpoint is not available
	assert.Equal(t, http.StatusServiceUnavailable, authrsp.StatusCode)
}

func TestAccessControl(t *testing.T) {
	skipUnlessEnabled(t)

	transport := (TestHTTPSClient.Transport).(*http.Transport)
	defer func(t time.Duration) {
		transport.ResponseHeaderTimeout = t
	}(transport.ResponseHeaderTimeout)
	transport.ResponseHeaderTimeout = 5 * time.Second

	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/projects":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		case "/api/v4/projects/1000":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		case "/api/v4/projects/2000":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
		case "/api/v4/projects/3000":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
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

	cases := []struct {
		Host         string
		Path         string
		Status       int
		RedirectBack bool
		Description  string
	}{
		{
			"group.gitlab-example.com",
			"/private.project/",
			http.StatusOK,
			false,
			"project with access",
		},
		{
			"group.gitlab-example.com",
			"/private.project.1/",
			http.StatusNotFound, // Do not expose project existed
			false,
			"project without access",
		},
		{
			"group.gitlab-example.com",
			"/private.project.2/",
			http.StatusFound,
			true,
			"invalid token test should redirect back",
		},
		{
			"group.gitlab-example.com",
			"/nonexistent/",
			http.StatusNotFound,
			false,
			"no project should redirect to login and then return 404",
		},
	}

	for _, c := range cases {

		t.Run(fmt.Sprintf("Access Control Test: %s", c.Description), func(t *testing.T) {
			teardown := RunPagesProcessWithAuthServerWithSSL(t, *pagesBinary, listeners, "", certFile, testServer.URL)
			defer teardown()

			rsp, err := GetRedirectPage(t, httpsListener, c.Host, c.Path)

			require.NoError(t, err)
			defer rsp.Body.Close()

			assert.Equal(t, http.StatusFound, rsp.StatusCode)

			cookie := rsp.Header.Get("Set-Cookie")

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpsListener, "gitlab-example.com", "/auth?code=1&state="+
				url.Query().Get("state"), cookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// server returns the ticket, user will be redirected to the project page
			assert.Equal(t, http.StatusFound, authrsp.StatusCode)
			cookie = authrsp.Header.Get("Set-Cookie")
			rsp, err = GetRedirectPageWithCookie(t, httpsListener, c.Host, c.Path, cookie)

			require.NoError(t, err)
			defer rsp.Body.Close()

			assert.Equal(t, c.Status, rsp.StatusCode)

			if c.RedirectBack {
				url, err = url.Parse(rsp.Header.Get("Location"))
				require.NoError(t, err)

				assert.Equal(t, "https", url.Scheme)
				assert.Equal(t, c.Host, url.Host)
				assert.Equal(t, c.Path, url.Path)
			}
		})
	}
}

func TestWhenLoginCallbackWithCorrectStateWithEndpointButTokenIsInvalid(t *testing.T) {
	skipUnlessEnabled(t)

	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/projects/1000":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetRedirectPageWithCookie(t, httpsListener, "gitlab-example.com", "/auth?code=1&state="+
		url.Query().Get("state"), cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// server returns the ticket, user will be redirected to the project page
	assert.Equal(t, http.StatusFound, authrsp.StatusCode)
	cookie = authrsp.Header.Get("Set-Cookie")
	rsp, err = GetRedirectPageWithCookie(t, httpsListener, "group.gitlab-example.com", "private.project/", cookie)

	require.NoError(t, err)
	defer rsp.Body.Close()

	// server returns token is invalid and removes token from cookie and redirects user back to be redirected for new token
	assert.Equal(t, http.StatusFound, rsp.StatusCode)
	url, err = url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	assert.Equal(t, "https", url.Scheme)
	assert.Equal(t, "group.gitlab-example.com", url.Host)
	assert.Equal(t, "/private.project/", url.Path)
}
