package main

import (
	"compress/gzip"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupServeHTTP(t *testing.T) {
	setUpTests()

	testGroup := &domain{
		Group:       "group",
		ProjectName: "",
	}

	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/", nil, "main-dir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/index.html", nil, "main-dir")
	assert.HTTPRedirect(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project", nil)
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project", nil,
		`<a href="//group.test.io/project/">Found</a>`)
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/", nil, "project-subdir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/index.html", nil, "project-subdir")
	assert.HTTPRedirect(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/subdir", nil)
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/subdir", nil,
		`<a href="//group.test.io/project/subdir/">Found</a>`)
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/subdir/", nil, "project-subsubdir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project2/", nil, "project2-main")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project2/index.html", nil, "project2-main")
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io//about.gitlab.com/%2e%2e", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink/index.html", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink/subdir/", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/fifo", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/not-existing-file", nil)
	assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project//about.gitlab.com/%2e%2e", nil)
}

func TestDomainServeHTTP(t *testing.T) {
	setUpTests()

	testDomain := &domain{
		Group:       "group",
		ProjectName: "project2",
		Config: &domainConfig{
			Domain: "test.domain.com",
		},
	}

	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/", nil, "project2-main")
	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/index.html", nil, "project2-main")
	assert.HTTPRedirect(t, testDomain.ServeHTTP, "GET", "/subdir", nil)
	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/subdir", nil,
		`<a href="/subdir/">Found</a>`)
	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/subdir/", nil, "project2-subdir")
	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/subdir/index.html", nil, "project2-subdir")
	assert.HTTPError(t, testDomain.ServeHTTP, "GET", "//about.gitlab.com/%2e%2e", nil)
	assert.HTTPError(t, testDomain.ServeHTTP, "GET", "/not-existing-file", nil)
}

func testHTTPGzip(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, acceptEncoding string, str interface{}, ungzip bool) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	if acceptEncoding != "" {
		req.Header.Add("Accept-Encoding", acceptEncoding)
	}
	handler(w, req)

	if ungzip {
		reader, err := gzip.NewReader(w.Body)
		require.NoError(t, err)
		defer reader.Close()

		contentEncoding := w.Header().Get("Content-Encoding")
		assert.Equal(t, "gzip", contentEncoding, "Content-Encoding")

		bytes, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		assert.Contains(t, string(bytes), str)
	} else {
		assert.Contains(t, w.Body.String(), str)
	}
}

func TestGroupServeHTTPGzip(t *testing.T) {
	setUpTests()

	testGroup := &domain{
		Group:       "group",
		ProjectName: "",
	}

	testSet := []struct {
		mode           string      // HTTP mode
		url            string      // Test URL
		params         url.Values  // Test URL params
		acceptEncoding string      // Accept encoding header
		body           interface{} // Expected body at above URL
		ungzip         bool        // Do we expect the request to require unzip?
	}{
		// No gzip encoding requested
		{"GET", "http://group.test.io/", nil, "", "main-dir", false},
		{"GET", "http://group.test.io/", nil, "identity", "main-dir", false},
		{"GET", "http://group.test.io/", nil, "gzip; q=0", "main-dir", false},
		// gzip encoding requeste},
		{"GET", "http://group.test.io/", nil, "*", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "identity, gzip", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip; q=1", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip; q=0.9", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip, deflate", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip; q=1, deflate", "main-dir", true},
		{"GET", "http://group.test.io/", nil, "gzip; q=0.9, deflate", "main-dir", true},
		// gzip encoding requested, but url does not have compressed content on disk
		{"GET", "http://group.test.io/project2/", nil, "*", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "identity, gzip", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip; q=1", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip; q=0.9", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip, deflate", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip; q=1, deflate", "project2-main", false},
		{"GET", "http://group.test.io/project2/", nil, "gzip; q=0.9, deflate", "project2-main", false},
		// malformed headers
		{"GET", "http://group.test.io/", nil, ";; gzip", "main-dir", false},
		{"GET", "http://group.test.io/", nil, "middle-out", "main-dir", false},
		{"GET", "http://group.test.io/", nil, "gzip; quality=1", "main-dir", false},
		// Symlinked .gz files are not supported
		{"GET", "http://group.test.io/gz-symlink", nil, "*", "data", false},
	}

	for _, tt := range testSet {
		testHTTPGzip(t, testGroup.ServeHTTP, tt.mode, tt.url, tt.params, tt.acceptEncoding, tt.body, tt.ungzip)
	}
}

func testHTTP404(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, str interface{}) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	handler(w, req)

	contentType, _, _ := mime.ParseMediaType(w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusNotFound, w.Code, "HTTP status")
	assert.Equal(t, "text/html", contentType, "Content-Type")
	assert.Contains(t, w.Body.String(), str)
}

func TestGroup404ServeHTTP(t *testing.T) {
	setUpTests()

	testGroup := &domain{
		Group:       "group.404",
		ProjectName: "",
	}

	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404/not/existing-file", nil, "Custom 404 project page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404/", nil, "Custom 404 project page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.no.404/not/existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/not/existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
	assert.HTTPBodyNotContains(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404.symlink/not/existing-file", nil, "Custom 404 project page")
}

func TestDomain404ServeHTTP(t *testing.T) {
	setUpTests()

	testDomain := &domain{
		Group:       "group.404",
		ProjectName: "domain.404",
		Config: &domainConfig{
			Domain: "domain.404.com",
		},
	}

	testHTTP404(t, testDomain.ServeHTTP, "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testDomain.ServeHTTP, "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
}

func TestPredefined404ServeHTTP(t *testing.T) {
	setUpTests()

	testDomain := &domain{
		Group: "group",
	}

	testHTTP404(t, testDomain.ServeHTTP, "GET", "http://group.test.io/not-existing-file", nil, "The page you're looking for could not be found")
}

func TestGroupCertificate(t *testing.T) {
	testGroup := &domain{
		Group:       "group",
		ProjectName: "",
	}

	tls, err := testGroup.ensureCertificate()
	assert.Nil(t, tls)
	assert.Error(t, err)
}

func TestDomainNoCertificate(t *testing.T) {
	testDomain := &domain{
		Group:       "group",
		ProjectName: "project2",
		Config: &domainConfig{
			Domain: "test.domain.com",
		},
	}

	tls, err := testDomain.ensureCertificate()
	assert.Nil(t, tls)
	assert.Error(t, err)

	_, err2 := testDomain.ensureCertificate()
	assert.Error(t, err)
	assert.Equal(t, err, err2)
}

func TestDomainCertificate(t *testing.T) {
	testDomain := &domain{
		Group:       "group",
		ProjectName: "project2",
		Config: &domainConfig{
			Domain:      "test.domain.com",
			Certificate: CertificateFixture,
			Key:         KeyFixture,
		},
	}

	tls, err := testDomain.ensureCertificate()
	assert.NotNil(t, tls)
	require.NoError(t, err)
}

func TestCacheControlHeaders(t *testing.T) {
	testGroup := &domain{Group: "group"}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://group.test.io/", nil)
	require.NoError(t, err)

	now := time.Now()
	testGroup.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "max-age=600", w.Header().Get("Cache-Control"))

	expires := w.Header().Get("Expires")
	require.NotEmpty(t, expires)

	expiresTime, err := time.Parse(time.RFC1123, expires)
	require.NoError(t, err)

	assert.WithinDuration(t, now.UTC().Add(10*time.Minute), expiresTime.UTC(), time.Minute)
}
