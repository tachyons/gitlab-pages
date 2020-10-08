package disk

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func serveFileOrNotFound(domain *domain.Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !domain.ServeFileHTTP(w, r) {
			domain.ServeNotFoundHTTP(w, r)
		}
	}
}

func testGroupServeHTTPHost(t *testing.T, host string) {
	t.Helper()

	testGroup := &domain.Domain{
		Resolver: &Group{
			name: "group",
			projects: map[string]*projectConfig{
				"group.test.io":            &projectConfig{},
				"group.gitlab-example.com": &projectConfig{},
				"project":                  &projectConfig{},
				"project2":                 &projectConfig{},
			},
		},
	}

	makeURL := func(path string) string {
		return "http://" + host + path
	}

	serve := serveFileOrNotFound(testGroup)

	require.HTTPBodyContains(t, serve, "GET", makeURL("/"), nil, "main-dir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/index"), nil, "main-dir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/index.html"), nil, "main-dir")
	testhelpers.AssertRedirectTo(t, serve, "GET", makeURL("/project"), nil, "//"+host+"/project/")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project/"), nil, "project-subdir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project/index"), nil, "project-subdir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project/index/"), nil, "project-subdir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project/index.html"), nil, "project-subdir")
	testhelpers.AssertRedirectTo(t, serve, "GET", makeURL("/project/subdir"), nil, "//"+host+"/project/subdir/")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project/subdir/"), nil, "project-subsubdir")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project2/"), nil, "project2-main")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project2/index"), nil, "project2-main")
	require.HTTPBodyContains(t, serve, "GET", makeURL("/project2/index.html"), nil, "project2-main")
	require.HTTPError(t, serve, "GET", makeURL("/private.project/"), nil)
	require.HTTPError(t, serve, "GET", makeURL("//about.gitlab.com/%2e%2e"), nil)
	require.HTTPError(t, serve, "GET", makeURL("/symlink"), nil)
	require.HTTPError(t, serve, "GET", makeURL("/symlink/index.html"), nil)
	require.HTTPError(t, serve, "GET", makeURL("/symlink/subdir/"), nil)
	require.HTTPError(t, serve, "GET", makeURL("/project/fifo"), nil)
	require.HTTPError(t, serve, "GET", makeURL("/not-existing-file"), nil)
	require.HTTPRedirect(t, serve, "GET", makeURL("/project//about.gitlab.com/%2e%2e"), nil)
}

func TestGroupServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	t.Run("group.test.io", func(t *testing.T) { testGroupServeHTTPHost(t, "group.test.io") })
	t.Run("group.test.io:8080", func(t *testing.T) { testGroupServeHTTPHost(t, "group.test.io:8080") })
}

func TestDomainServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &domain.Domain{
		Name: "test.domain.com",
		Resolver: &customProjectResolver{
			path:   "group/project2/public",
			config: &domainConfig{},
		},
	}

	require.HTTPBodyContains(t, serveFileOrNotFound(testDomain), "GET", "/", nil, "project2-main")
	require.HTTPBodyContains(t, serveFileOrNotFound(testDomain), "GET", "/index.html", nil, "project2-main")
	require.HTTPRedirect(t, serveFileOrNotFound(testDomain), "GET", "/subdir", nil)
	require.HTTPBodyContains(t, serveFileOrNotFound(testDomain), "GET", "/subdir", nil,
		`<a href="/subdir/">Found</a>`)
	require.HTTPBodyContains(t, serveFileOrNotFound(testDomain), "GET", "/subdir/", nil, "project2-subdir")
	require.HTTPBodyContains(t, serveFileOrNotFound(testDomain), "GET", "/subdir/index.html", nil, "project2-subdir")
	require.HTTPError(t, serveFileOrNotFound(testDomain), "GET", "//about.gitlab.com/%2e%2e", nil)
	require.HTTPError(t, serveFileOrNotFound(testDomain), "GET", "/not-existing-file", nil)
}

func TestIsHTTPSOnly(t *testing.T) {
	tests := []struct {
		name     string
		domain   *domain.Domain
		url      string
		expected bool
	}{
		{
			name: "Default group domain with HTTPS-only enabled",
			domain: &domain.Domain{
				Resolver: &Group{
					name:     "group",
					projects: projects{"test-domain": &projectConfig{HTTPSOnly: true}},
				},
			},
			url:      "http://test-domain",
			expected: true,
		},
		{
			name: "Default group domain with HTTPS-only disabled",
			domain: &domain.Domain{
				Resolver: &Group{
					name:     "group",
					projects: projects{"test-domain": &projectConfig{HTTPSOnly: false}},
				},
			},
			url:      "http://test-domain",
			expected: false,
		},
		{
			name: "Case-insensitive default group domain with HTTPS-only enabled",
			domain: &domain.Domain{
				Resolver: &Group{
					name:     "group",
					projects: projects{"test-domain": &projectConfig{HTTPSOnly: true}},
				},
			},
			url:      "http://Test-domain",
			expected: true,
		},
		{
			name: "Other group domain with HTTPS-only enabled",
			domain: &domain.Domain{
				Resolver: &Group{
					name:     "group",
					projects: projects{"project": &projectConfig{HTTPSOnly: true}},
				},
			},
			url:      "http://test-domain/project",
			expected: true,
		},
		{
			name: "Other group domain with HTTPS-only disabled",
			domain: &domain.Domain{
				Resolver: &Group{
					name:     "group",
					projects: projects{"project": &projectConfig{HTTPSOnly: false}},
				},
			},
			url:      "http://test-domain/project",
			expected: false,
		},
		{
			name:     "Unknown project",
			domain:   &domain.Domain{},
			url:      "http://test-domain/project",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, test.url, nil)
			require.Equal(t, test.expected, test.domain.IsHTTPSOnly(req))
		})
	}
}

func testHTTPGzip(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, acceptEncoding string, str interface{}, contentType string, expectCompressed bool) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	if acceptEncoding != "" {
		req.Header.Add("Accept-Encoding", acceptEncoding)
	}
	handler(w, req)

	if expectCompressed {
		contentLength := w.Header().Get("Content-Length")
		require.Equal(t, strconv.Itoa(w.Body.Len()), contentLength, "Content-Length")

		contentEncoding := w.Header().Get("Content-Encoding")
		require.Equal(t, "gzip", contentEncoding, "Content-Encoding")

		reader, err := gzip.NewReader(w.Body)
		require.NoError(t, err)
		defer reader.Close()

		bytes, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		require.Contains(t, string(bytes), str)
	} else {
		require.Contains(t, w.Body.String(), str)
	}

	require.Equal(t, contentType, w.Header().Get("Content-Type"))
}

func TestGroupServeHTTPGzip(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testGroup := &domain.Domain{
		Resolver: &Group{
			name: "group",
			projects: map[string]*projectConfig{
				"group.test.io":            &projectConfig{},
				"group.gitlab-example.com": &projectConfig{},
				"project":                  &projectConfig{},
				"project2":                 &projectConfig{},
			},
		},
	}

	testSet := []struct {
		mode             string      // HTTP mode
		url              string      // Test URL
		acceptEncoding   string      // Accept encoding header
		body             interface{} // Expected body at above URL
		contentType      string      // Expected content-type
		expectCompressed bool        // Expect the response to be gzipped?
	}{
		// No gzip encoding requested
		{"GET", "/index.html", "", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "identity", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "gzip; q=0", "main-dir", "text/html; charset=utf-8", false},
		// gzip encoding requested,
		{"GET", "/index.html", "identity, gzip", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip; q=1", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip; q=0.9", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip; q=1, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip; q=0.9, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=0.9, gzip; q=1", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=0, gzip; q=1", "main-dir", "text/html; charset=utf-8", true},
		// fallback to gzip because .br is missing
		{"GET", "/index2.html", "*", "main-dir", "text/html; charset=utf-8", true},
		// gzip encoding requested, but url does not have compressed content on disk
		{"GET", "/project2/index.html", "*", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "identity, gzip", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip; q=1", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip; q=0.9", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip, deflate", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip; q=1, deflate", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "gzip; q=0.9, deflate", "project2-main", "text/html; charset=utf-8", false},
		// malformed headers
		{"GET", "/index.html", ";; gzip", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "middle-out", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "gzip; quality=1", "main-dir", "text/html; charset=utf-8", false},
		// Symlinked .gz files are not supported
		{"GET", "/gz-symlink", "*", "data", "text/plain; charset=utf-8", false},
		// Unknown file-extension, with text content
		{"GET", "/text.unknown", "gzip", "hello", "text/plain; charset=utf-8", true},
		{"GET", "/text-nogzip.unknown", "*", "hello", "text/plain; charset=utf-8", false},
		// Unknown file-extension, with PNG content
		{"GET", "/image.unknown", "gzip", "GIF89a", "image/gif", true},
		{"GET", "/image-nogzip.unknown", "*", "GIF89a", "image/gif", false},
	}

	for _, tt := range testSet {
		t.Run(tt.url+" acceptEncoding: "+tt.acceptEncoding, func(t *testing.T) {
			URL := "http://group.test.io" + tt.url
			testHTTPGzip(t, serveFileOrNotFound(testGroup), tt.mode, URL, nil, tt.acceptEncoding, tt.body, tt.contentType, tt.expectCompressed)
		})
	}
}

func testHTTPBrotli(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, acceptEncoding string, str interface{}, contentType string, expectCompressed bool) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	if acceptEncoding != "" {
		req.Header.Add("Accept-Encoding", acceptEncoding)
	}
	handler(w, req)

	if expectCompressed {
		contentLength := w.Header().Get("Content-Length")
		require.Equal(t, strconv.Itoa(w.Body.Len()), contentLength, "Content-Length")

		contentEncoding := w.Header().Get("Content-Encoding")
		require.Equal(t, "br", contentEncoding, "Content-Encoding")

		reader := brotli.NewReader(w.Body)
		bytes, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		require.Contains(t, string(bytes), str)
	} else {
		require.Contains(t, w.Body.String(), str)
	}

	require.Equal(t, contentType, w.Header().Get("Content-Type"))
}

func TestGroupServeHTTPBrotli(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testGroup := &domain.Domain{
		Resolver: &Group{
			name: "group",
			projects: map[string]*projectConfig{
				"group.test.io":            &projectConfig{},
				"group.gitlab-example.com": &projectConfig{},
				"project":                  &projectConfig{},
				"project2":                 &projectConfig{},
			},
		},
	}

	testSet := []struct {
		mode             string      // HTTP mode
		url              string      // Test URL
		acceptEncoding   string      // Accept encoding header
		body             interface{} // Expected body at above URL
		contentType      string      // Expected content-type
		expectCompressed bool        // Expect the response to be br compressed?
	}{
		// No br encoding requested
		{"GET", "/index.html", "", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "identity", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "br; q=0", "main-dir", "text/html; charset=utf-8", false},
		// br encoding requested,
		{"GET", "/index.html", "*", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "identity, br", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=1", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=0.9", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=1, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "br; q=0.9, deflate", "main-dir", "text/html; charset=utf-8", true},
		{"GET", "/index.html", "gzip; q=0.5, br; q=1", "main-dir", "text/html; charset=utf-8", true},
		// br encoding requested, but url does not have compressed content on disk
		{"GET", "/project2/index.html", "*", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "identity, br", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br; q=1", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br; q=0.9", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br, deflate", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br; q=1, deflate", "project2-main", "text/html; charset=utf-8", false},
		{"GET", "/project2/index.html", "br; q=0.9, deflate", "project2-main", "text/html; charset=utf-8", false},
		// malformed headers
		{"GET", "/index.html", ";; br", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "middle-out", "main-dir", "text/html; charset=utf-8", false},
		{"GET", "/index.html", "br; quality=1", "main-dir", "text/html; charset=utf-8", false},
		// Symlinked .br files are not supported
		{"GET", "/gz-symlink", "*", "data", "text/plain; charset=utf-8", false},
		// Unknown file-extension, with text content
		{"GET", "/text.unknown", "*", "hello", "text/plain; charset=utf-8", true},
		{"GET", "/text-nogzip.unknown", "*", "hello", "text/plain; charset=utf-8", false},
		// Unknown file-extension, with PNG content
		{"GET", "/image.unknown", "*", "GIF89a", "image/gif", true},
		{"GET", "/image-nogzip.unknown", "*", "GIF89a", "image/gif", false},
	}

	for _, tt := range testSet {
		t.Run(tt.url+" acceptEncoding: "+tt.acceptEncoding, func(t *testing.T) {
			URL := "http://group.test.io" + tt.url
			testHTTPBrotli(t, serveFileOrNotFound(testGroup), tt.mode, URL, nil, tt.acceptEncoding, tt.body, tt.contentType, tt.expectCompressed)
		})
	}
}

func TestGroup404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testGroup := &domain.Domain{
		Resolver: &Group{
			name: "group.404",
			projects: map[string]*projectConfig{
				"domain.404":          &projectConfig{},
				"group.404.test.io":   &projectConfig{},
				"project.404":         &projectConfig{},
				"project.404.symlink": &projectConfig{},
				"project.no.404":      &projectConfig{},
			},
		},
	}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/project.404/not/existing-file", nil, "Custom 404 project page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/project.404/", nil, "Custom 404 project page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/not/existing-file", nil, "Custom 404 group page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
	require.HTTPBodyNotContains(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/project.404.symlink/not/existing-file", nil, "Custom 404 project page")

	// Ensure the namespace project's custom 404.html is not used by projects
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testGroup), "GET", "http://group.404.test.io/project.no.404/not/existing-file", nil, "The page you're looking for could not be found.")
}

func TestDomain404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &domain.Domain{
		Resolver: &customProjectResolver{
			path:   "group.404/domain.404/public/",
			config: &domainConfig{Domain: "domain.404.com"},
		},
	}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.404.test.io/not-existing-file", nil, "Custom domain.404 page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.404.test.io/", nil, "Custom domain.404 page")
}

func TestPredefined404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &domain.Domain{}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.test.io/not-existing-file", nil, "The page you're looking for could not be found")
}

func TestGroupCertificate(t *testing.T) {
	testGroup := &domain.Domain{}

	tls, err := testGroup.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)
}

func TestDomainNoCertificate(t *testing.T) {
	testDomain := &domain.Domain{
		Resolver: &customProjectResolver{
			path:   "group/project2/public",
			config: &domainConfig{Domain: "test.domain.com"},
		},
	}

	tls, err := testDomain.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)

	_, err2 := testDomain.EnsureCertificate()
	require.Error(t, err)
	require.Equal(t, err, err2)
}

func TestDomainCertificate(t *testing.T) {
	testDomain := &domain.Domain{
		Name:            "test.domain.com",
		CertificateCert: fixture.Certificate,
		CertificateKey:  fixture.Key,
		Resolver: &customProjectResolver{
			path: "group/project2/public",
		},
	}

	tls, err := testDomain.EnsureCertificate()
	require.NotNil(t, tls)
	require.NoError(t, err)
}

func TestCacheControlHeaders(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testGroup := &domain.Domain{
		Resolver: &Group{
			name: "group",
			projects: map[string]*projectConfig{
				"group.test.io": &projectConfig{},
			},
		},
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://group.test.io/", nil)
	require.NoError(t, err)

	now := time.Now()
	serveFileOrNotFound(testGroup)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "max-age=600", w.Header().Get("Cache-Control"))

	expires := w.Header().Get("Expires")
	require.NotEmpty(t, expires)

	expiresTime, err := time.Parse(time.RFC1123, expires)
	require.NoError(t, err)

	require.WithinDuration(t, now.UTC().Add(10*time.Minute), expiresTime.UTC(), time.Minute)
}

var chdirSet = false

func setUpTests(t *testing.T) func() {
	t.Helper()
	return testhelpers.ChdirInPath(t, "../../../shared/pages", &chdirSet)
}
