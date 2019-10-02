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

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

type stubbedResolver struct {
	project *serving.LookupPath
	subpath string
	err     error
}

func (resolver *stubbedResolver) Resolve(*http.Request) (*serving.LookupPath, string, error) {
	return resolver.project, resolver.subpath, resolver.err
}

func serveFileOrNotFound(domain *Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !domain.ServeFileHTTP(w, r) {
			domain.ServeNotFoundHTTP(w, r)
		}
	}
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
				Resolver: &stubbedResolver{
					project: &serving.LookupPath{
						Path:        "group/project/public",
						IsHTTPSOnly: true,
					},
				},
			},
			url:      "http://custom-domain",
			expected: true,
		},
		{
			name: "Custom domain with HTTPS-only disabled",
			domain: &Domain{
				Resolver: &stubbedResolver{
					project: &serving.LookupPath{
						Path:        "group/project/public",
						IsHTTPSOnly: false,
					},
				},
			},
			url:      "http://custom-domain",
			expected: false,
		},
		{
			name:     "Unknown project",
			domain:   &Domain{},
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

func TestPredefined404ServeHTTP(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	testDomain := &Domain{}

	testhelpers.AssertHTTP404(t, serveFileOrNotFound(testDomain), "GET", "http://group.test.io/not-existing-file", nil, "The page you're looking for could not be found")
}

func TestGroupCertificate(t *testing.T) {
	testGroup := &Domain{}

	tls, err := testGroup.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)
}

func TestDomainNoCertificate(t *testing.T) {
	testDomain := &Domain{
		Name: "test.domain.com",
	}

	tls, err := testDomain.EnsureCertificate()
	require.Nil(t, tls)
	require.Error(t, err)

	_, err2 := testDomain.EnsureCertificate()
	require.Error(t, err)
	require.Equal(t, err, err2)
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
