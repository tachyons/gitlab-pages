package domain

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func serveFileOrNotFound(domain *Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !domain.ServeFileHTTP(w, r) {
			domain.ServeNotFoundHTTP(w, r)
		}
	}
}

func TestDomainServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &Domain{
		Project:       "project2",
		Group:         "group",
		ProjectConfig: &ProjectConfig{DomainName: "test.domain.com"},
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
		domain   *Domain
		url      string
		expected bool
	}{
		{
			name: "Custom domain with HTTPS-only enabled",
			domain: &Domain{
				Project:       "project",
				Group:         "group",
				ProjectConfig: &ProjectConfig{HTTPSOnly: true},
			},
			url:      "http://custom-domain",
			expected: true,
		},
		{
			name: "Custom domain with HTTPS-only disabled",
			domain: &Domain{
				Project:       "project",
				Group:         "group",
				ProjectConfig: &ProjectConfig{HTTPSOnly: false},
			},
			url:      "http://custom-domain",
			expected: false,
		},
		{
			name: "Unknown project",
			domain: &Domain{
				Project: "project",
				Group:   "group",
			},
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

func TestHasAcmeChallenge(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	tests := []struct {
		name     string
		domain   *Domain
		token    string
		expected bool
	}{
		{
			name: "Project containing acme challenge",
			domain: &Domain{
				Group:         "group.acme",
				Project:       "with.acme.challenge",
				ProjectConfig: &ProjectConfig{HTTPSOnly: true},
			},
			token:    "existingtoken",
			expected: true,
		},
		{
			name: "Project containing acme challenge",
			domain: &Domain{
				Group:         "group.acme",
				Project:       "with.acme.challenge",
				ProjectConfig: &ProjectConfig{HTTPSOnly: true},
			},
			token:    "foldertoken",
			expected: true,
		},
		{
			name: "Project containing another token",
			domain: &Domain{
				Group:         "group.acme",
				Project:       "with.acme.challenge",
				ProjectConfig: &ProjectConfig{HTTPSOnly: true},
			},
			token:    "notexistingtoken",
			expected: false,
		},
		{
			name:     "nil domain",
			domain:   nil,
			token:    "existingtoken",
			expected: false,
		},
		{
			name: "Domain without config",
			domain: &Domain{
				Group:   "group.acme",
				Project: "with.acme.challenge",
			},
			token:    "existingtoken",
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.domain.HasAcmeChallenge(test.token))
		})
	}
}

func testHTTPGzip(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, acceptEncoding string, str interface{}, contentType string, ungzip bool) {
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
		require.Equal(t, "gzip", contentEncoding, "Content-Encoding")

		bytes, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		require.Contains(t, string(bytes), str)
	} else {
		require.Contains(t, w.Body.String(), str)
	}

	require.Equal(t, contentType, w.Header().Get("Content-Type"))
}

func TestDomain404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &Domain{
		Group:         "group.404",
		Project:       "domain.404",
		ProjectConfig: &ProjectConfig{DomainName: "domain.404.com"},
	}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
}

func TestPredefined404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &Domain{
		Group: "group",
	}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.test.io/not-existing-file", nil, "The page you're looking for could not be found")
}

func TestGroupCertificate(t *testing.T) {
	testGroup := &Domain{
		Project: "",
		Group:   "group",
	}

	tls, err := testGroup.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)
}

func TestDomainNoCertificate(t *testing.T) {
	testDomain := &Domain{
		Group:         "group",
		Project:       "project2",
		ProjectConfig: &ProjectConfig{DomainName: "test.domain.com"},
	}

	tls, err := testDomain.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)

	_, err2 := testDomain.EnsureCertificate()
	require.Error(t, err)
	require.Equal(t, err, err2)
}

func TestDomainCertificate(t *testing.T) {
	testDomain := &Domain{
		Group:   "group",
		Project: "project2",
		ProjectConfig: &ProjectConfig{
			DomainName:  "test.domain.com",
			Certificate: fixture.Certificate,
			Key:         fixture.Key,
		},
	}

	tls, err := testDomain.EnsureCertificate()
	require.NotNil(t, tls)
	require.NoError(t, err)
}

var chdirSet = false

func setUpTests(t require.TestingT) func() {
	return chdirInPath(t, "../../shared/pages")
}

func chdirInPath(t require.TestingT, path string) func() {
	noOp := func() {}
	if chdirSet {
		return noOp
	}

	cwd, err := os.Getwd()
	require.NoError(t, err, "Cannot Getwd")

	err = os.Chdir(path)
	require.NoError(t, err, "Cannot Chdir")

	chdirSet = true
	return func() {
		err := os.Chdir(cwd)
		require.NoError(t, err, "Cannot Chdir in cleanup")

		chdirSet = false
	}
}
