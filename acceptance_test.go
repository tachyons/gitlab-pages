package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
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

func skipUnlessEnabled(t *testing.T) {
	if testing.Short() {
		t.Log("Acceptance tests disabled")
		t.SkipNow()
	}

	if _, err := os.Stat(*pagesBinary); os.IsNotExist(err) {
		t.Errorf("Couldn't find gitlab-pages binary at %s", *pagesBinary)
		t.FailNow()
	}
}

func TestUnknownHostReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=false")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "invalid.invalid", "")

		assert.NoError(t, err)
		rsp.Body.Close()
		assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestKnownHostReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "project/")

		assert.NoError(t, err)
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

	rsp, err := DoPagesRequest(t, req)
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

		assert.NoError(t, err)
		rsp.Body.Close()
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}

}

func TestHttpToHttpsRedirectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=false")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpToHttpsRedirectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=true")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)
	assert.Equal(t, 1, len(rsp.Header["Location"]))
	assert.Equal(t, "https://group.gitlab-example.com/project/", rsp.Header.Get("Location"))

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	assert.NoError(t, err)
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
		assert.Contains(t, string(body), "gitlab_pages_domains_served_total 7")
	}
}

func TestStatusPage(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=false", "-pages-status=/@statuscheck")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestStatusNotYetReady(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=false", "-pages-status=/@statuscheck", "-pages-root=shared/invalid-pages")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestPageNotAvailableIfNotLoaded(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=false", "-pages-root=shared/invalid-pages")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html")
	assert.NoError(t, err)
	defer rsp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, rsp.StatusCode)
}

func TestArtifactProxyRequest(t *testing.T) {
	skipUnlessEnabled(t)
	content := "<!DOCTYPE html><html><head><title>Title of the document</title></head><body></body></html>"
	contentLength := int64(len(content))
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	for _, c := range cases {
		t.Run(fmt.Sprintf("Proxy Request Test: %s", c.Description), func(t *testing.T) {
			teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-artifacts-server="+testServer.URL+"/api/v4", c.BinaryOption)
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

	teardown := RunPagesProcess(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com:", "project/")

	assert.NoError(t, err)
	rsp.Body.Close()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	skipUnlessEnabled(t)
	os.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcess(t, *pagesBinary, []ListenSpec{httpsListener}, "")
	defer teardown()

	for _, listener := range []ListenSpec{httpListener, httpsListener} {
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		assert.NoError(t, err)
		rsp.Body.Close()
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestMultiFlagEnvironmentVariables(t *testing.T) {
	skipUnlessEnabled(t)
	listenSpec := []ListenSpec{{"http", "127.0.0.1", "37001"}, {"http", "127.0.0.1", "37002"}}
	envVarValue := fmt.Sprintf("%s,%s", net.JoinHostPort("127.0.0.1", "37001"), net.JoinHostPort("127.0.0.1", "37002"))

	os.Setenv("LISTEN_HTTP", envVarValue)
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcess(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()

	for _, listener := range listenSpec {
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		assert.NoError(t, err)
		rsp.Body.Close()
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}
