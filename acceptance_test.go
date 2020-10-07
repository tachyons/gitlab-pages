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

const (
	objectStorageMockServer = "127.0.0.1:37003"
)

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

func TestCustom404(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	tests := []struct {
		host    string
		path    string
		content string
	}{
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.404/not/existing-file",
			content: "Custom 404 project page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.404/",
			content: "Custom 404 project page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "not/existing-file",
			content: "Custom 404 group page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "not-existing-file",
			content: "Custom 404 group page",
		},
		{
			host:    "group.404.gitlab-example.com",
			content: "Custom 404 group page",
		},
		{
			host:    "domain.404.com",
			content: "Custom domain.404 page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.no.404/not/existing-file",
			content: "The page you're looking for could not be found.",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s/%s", test.host, test.path), func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, test.host, test.path)

				require.NoError(t, err)
				defer rsp.Body.Close()
				require.Equal(t, http.StatusNotFound, rsp.StatusCode)

				page, err := ioutil.ReadAll(rsp.Body)
				require.NoError(t, err)
				require.Contains(t, string(page), test.content)
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

	_, cleanup := newZipFileServerURL(t, "shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	teardown := RunPagesProcessWithStubGitLabServer(t, true, *pagesBinary, listeners, ":42345", []string{})
	defer teardown()

	// need to call an actual resource to populate certain metrics e.g. gitlab_pages_domains_source_api_requests_total
	res, err := GetPageFromListener(t, httpListener, "zip.gitlab.io", "/index.html/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	resp, err := http.Get("http://localhost:42345/metrics")
	require.NoError(t, err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "gitlab_pages_http_in_flight_requests 0")
	// TODO: remove metrics for disk source https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
	require.Contains(t, string(body), "gitlab_pages_served_domains 0")
	require.Contains(t, string(body), "gitlab_pages_domains_failed_total 0")
	require.Contains(t, string(body), "gitlab_pages_domains_updated_total 0")
	require.Contains(t, string(body), "gitlab_pages_last_domain_update_seconds gauge")
	require.Contains(t, string(body), "gitlab_pages_domains_configuration_update_duration gauge")
	// end TODO
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_hit")
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_miss")
	require.Contains(t, string(body), "gitlab_pages_domains_source_failures_total")
	require.Contains(t, string(body), "gitlab_pages_serverless_requests 0")
	require.Contains(t, string(body), "gitlab_pages_serverless_latency_sum 0")
	require.Contains(t, string(body), "gitlab_pages_disk_serving_file_size_bytes_sum")
	require.Contains(t, string(body), "gitlab_pages_serving_time_seconds_sum")
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_requests_total{status_code="200"}`)
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_call_duration_bucket`)
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_trace_duration`)
	// httprange
	require.Contains(t, string(body), `gitlab_pages_httprange_requests_total{status_code="206"}`)
	require.Contains(t, string(body), "gitlab_pages_httprange_requests_duration_bucket")
	require.Contains(t, string(body), "gitlab_pages_httprange_trace_duration")
	require.Contains(t, string(body), "gitlab_pages_httprange_open_requests")
	// zip archives
	require.Contains(t, string(body), "gitlab_pages_zip_opened")
	require.Contains(t, string(body), "gitlab_pages_zip_cache_requests")
	require.Contains(t, string(body), "gitlab_pages_zip_cached_archives")
	require.Contains(t, string(body), "gitlab_pages_zip_archive_entries_cached")
	require.Contains(t, string(body), "gitlab_pages_zip_opened_entries_count")
}

func TestDisabledRedirects(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, listeners, "", []string{"FF_ENABLE_REDIRECTS=false"})
	defer teardown()

	// Test that redirects status page is forbidden
	rsp, err := GetPageFromListener(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/_redirects")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusForbidden, rsp.StatusCode)

	// Test that redirects are disabled
	rsp, err = GetRedirectPage(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/redirect-portal.html")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestRedirectStatusPage(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/_redirects")
	require.NoError(t, err)

	body, err := ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Contains(t, string(body), "11 rules")
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestRedirect(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	// Test that serving a file still works with redirects enabled
	rsp, err := GetRedirectPage(t, httpListener, "group.redirects.gitlab-example.com", "/project-redirects/index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusOK, rsp.StatusCode)

	tests := []struct {
		host             string
		path             string
		expectedStatus   int
		expectedLocation string
	}{
		// Project domain
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/project-redirects/magic-land.html",
		},
		// Make sure invalid rule does not redirect
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/goto-domain.html",
			expectedStatus:   http.StatusNotFound,
			expectedLocation: "",
		},
		// Actual file on disk should override any redirects that match
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/project-redirects/file-override.html",
			expectedStatus:   http.StatusOK,
			expectedLocation: "",
		},
		// Group-level domain
		{
			host:             "group.redirects.gitlab-example.com",
			path:             "/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/magic-land.html",
		},
		// Custom domain
		{
			host:             "redirects.custom-domain.com",
			path:             "/redirect-portal.html",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/magic-land.html",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s%s -> %s (%d)", tt.host, tt.path, tt.expectedLocation, tt.expectedStatus), func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.host, tt.path)
			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, tt.expectedLocation, rsp.Header.Get("Location"))
			require.Equal(t, tt.expectedStatus, rsp.StatusCode)
		})
	}
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
			binaryOption: "artifacts-server-timeout=1",
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
			configFile, cleanup := defaultConfigFileWith(t,
				"artifacts-server="+artifactServerURL,
				"auth-server="+testServer.URL,
				"auth-redirect-uri=https://projects.gitlab-example.com/auth",
				tt.binaryOption)
			defer cleanup()

			teardown := RunPagesProcessWithSSLCertFile(
				t,
				*pagesBinary,
				listeners,
				"",
				certFile,
				"-config="+configFile,
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
			require.NoError(t, err)

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
	require.NoError(t, err)

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))

	url, err = url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	require.Equal(t, "https", url.Scheme)
	require.Equal(t, "gitlab-auth.com", url.Host)
	require.Equal(t, "/oauth/authorize", url.Path)
	require.Equal(t, "clientID", url.Query().Get("client_id"))
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
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestCustomErrorPageWithAuth(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")
	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

	tests := []struct {
		name              string
		domain            string
		path              string
		expectedErrorPage string
	}{
		{
			name:              "private_project_authorized",
			domain:            "group.404.gitlab-example.com",
			path:              "/private_project/unknown",
			expectedErrorPage: "Private custom 404 error page",
		},
		{
			name:   "public_namespace_with_private_unauthorized_project",
			domain: "group.404.gitlab-example.com",
			// /private_unauthorized/config.json resolves project ID to 2000 which will cause a 401 from the mock GitLab testServer
			path:              "/private_unauthorized/unknown",
			expectedErrorPage: "Custom 404 group page",
		},
		{
			name:              "private_namespace_authorized",
			domain:            "group.auth.gitlab-example.com",
			path:              "/unknown",
			expectedErrorPage: "group.auth.gitlab-example.com namespace custom 404",
		},
		{
			name:   "private_namespace_with_private_project_auth_failed",
			domain: "group.auth.gitlab-example.com",
			// project ID is 2000
			path:              "/private.project.1/unknown",
			expectedErrorPage: "The page you're looking for could not be found.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.domain, tt.path)
			require.NoError(t, err)
			defer rsp.Body.Close()

			cookie := rsp.Header.Get("Set-Cookie")

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			state := url.Query().Get("state")
			require.Equal(t, "http://"+tt.domain, url.Query().Get("domain"))

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
			require.Equal(t, tt.domain, url.Host)
			require.Equal(t, "1", url.Query().Get("code"))
			require.Equal(t, state, url.Query().Get("state"))

			// Run auth callback in custom domain
			authrsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code=1&state="+
				state, cookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// Will redirect to the page
			groupCookie := authrsp.Header.Get("Set-Cookie")
			require.Equal(t, http.StatusFound, authrsp.StatusCode)

			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain error page
			require.Equal(t, "http://"+tt.domain+tt.path, url.String())

			// Fetch page in custom domain
			anotherResp, err := GetRedirectPageWithCookie(t, httpListener, tt.domain, tt.path, groupCookie)
			require.NoError(t, err)

			require.Equal(t, http.StatusNotFound, anotherResp.StatusCode)

			page, err := ioutil.ReadAll(anotherResp.Body)
			require.NoError(t, err)
			require.Contains(t, string(page), tt.expectedErrorPage)
		})
	}
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
	require.NoError(t, err)
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

type runPagesFunc func(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, authServer string) func()

func testAccessControl(t *testing.T, runPages runPagesFunc) {
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
			teardown := runPages(t, *pagesBinary, listeners, "", certFile, testServer.URL)
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
			require.NoError(t, err)

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

func TestAccessControlWithSSLCertFile(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithAuthServerWithSSLCertFile)
}

func TestAccessControlWithSSLCertDir(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithAuthServerWithSSLCertDir)
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

func TestDomainsSource(t *testing.T) {
	skipUnlessEnabled(t)

	type args struct {
		configSource string
		domain       string
		urlSuffix    string
	}
	type want struct {
		statusCode int
		content    string
		apiCalled  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "gitlab_source_domain_exists",
			args: args{
				configSource: "gitlab",
				domain:       "new-source-test.gitlab.io",
				urlSuffix:    "/my/pages/project/",
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "New Pages GitLab Source TEST OK\n",
				apiCalled:  true,
			},
		},
		{
			name: "gitlab_source_domain_does_not_exist",
			args: args{
				configSource: "gitlab",
				domain:       "non-existent-domain.gitlab.io",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  true,
			},
		},
		{
			name: "disk_source_domain_exists",
			args: args{
				configSource: "disk",
				// test.domain.com sourced from disk configuration
				domain:    "test.domain.com",
				urlSuffix: "/",
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "main-dir\n",
				apiCalled:  false,
			},
		},
		{
			name: "disk_source_domain_does_not_exist",
			args: args{
				configSource: "disk",
				domain:       "non-existent-domain.gitlab.io",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  false,
			},
		},
		{
			name: "disk_source_domain_should_not_exist_under_hashed_dir",
			args: args{
				configSource: "disk",
				domain:       "hashed.com",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  false,
			},
		},
		// TODO: modify mock so we can test domain-config-source=auto when API/disk is not ready https://gitlab.com/gitlab-org/gitlab/-/issues/218358
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiCalled bool
			source := NewGitlabDomainsSourceStub(t, &apiCalled)
			defer source.Close()

			gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

			pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", tt.args.configSource}
			teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, listeners, "", []string{}, pagesArgs...)
			defer teardown()

			response, err := GetPageFromListener(t, httpListener, tt.args.domain, tt.args.urlSuffix)
			require.NoError(t, err)

			require.Equal(t, tt.want.statusCode, response.StatusCode)
			if tt.want.statusCode == http.StatusOK {
				defer response.Body.Close()
				body, err := ioutil.ReadAll(response.Body)
				require.NoError(t, err)

				require.Equal(t, tt.want.content, string(body), "content mismatch")
			}

			require.Equal(t, tt.want.apiCalled, apiCalled, "api called mismatch")
		})
	}
}

func TestZipServing(t *testing.T) {
	skipUnlessEnabled(t)

	var apiCalled bool
	source := NewGitlabDomainsSourceStub(t, &apiCalled)
	defer source.Close()

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

	pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", "gitlab"}
	teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, listeners, "", []string{}, pagesArgs...)
	defer teardown()

	_, cleanup := newZipFileServerURL(t, "shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	tests := map[string]struct {
		urlSuffix          string
		expectedStatusCode int
		expectedContent    string
	}{
		"base_domain_no_suffix": {
			urlSuffix:          "/",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists": {
			urlSuffix:          "/index.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/index.html\n",
		},
		"file_exists_in_subdir": {
			urlSuffix:          "/subdir/hello.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "zip.gitlab.io/project/subdir/hello.html\n",
		},
		"file_exists_symlink": {
			urlSuffix:          "/symlink.html",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "symlink.html->subdir/linked.html\n",
		},
		"dir": {
			urlSuffix:          "/subdir/",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"file_does_not_exist": {
			urlSuffix:          "/unknown.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
		"bad_symlink": {
			urlSuffix:          "/bad-symlink.html",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "zip.gitlab.io/project/404.html\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := GetPageFromListener(t, httpListener, "zip.gitlab.io", tt.urlSuffix)
			require.NoError(t, err)
			defer response.Body.Close()

			require.Equal(t, tt.expectedStatusCode, response.StatusCode)
			if tt.expectedStatusCode == http.StatusOK || tt.expectedStatusCode == http.StatusNotFound {
				body, err := ioutil.ReadAll(response.Body)
				require.NoError(t, err)

				require.Equal(t, tt.expectedContent, string(body), "content mismatch")
			}
		})
	}
}
