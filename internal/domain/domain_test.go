package domain

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

type stubbedResolver struct {
	project *serving.LookupPath
	subpath string
	err     error
}

func (resolver *stubbedResolver) Resolve(*http.Request) (*serving.Request, error) {
	return &serving.Request{
		Serving:    disk.New(),
		LookupPath: resolver.project,
		SubPath:    resolver.subpath,
	}, resolver.err
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

func BenchmarkEnsureCertificate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testDomain := &Domain{
			Name:            "test.domain.com",
			CertificateCert: fixture.Certificate,
			CertificateKey:  fixture.Key,
		}

		testDomain.EnsureCertificate()
	}
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

func TestServeNamespaceNotFound(t *testing.T) {
	tests := []struct {
		name             string
		domain           string
		path             string
		resolver         *stubbedResolver
		expectedResponse string
	}{
		{
			name:   "public_namespace_domain",
			domain: "group.404.gitlab-example.com",
			path:   "/unknown",
			resolver: &stubbedResolver{
				project: &serving.LookupPath{
					Path:               "../../shared/pages/group.404/group.404.gitlab-example.com/public",
					IsNamespaceProject: true,
				},
				subpath: "/unknown",
			},
			expectedResponse: "Custom 404 group page",
		},
		{
			name:   "private_project_under_public_namespace_domain",
			domain: "group.404.gitlab-example.com",
			path:   "/private_project/unknown",
			resolver: &stubbedResolver{
				project: &serving.LookupPath{
					Path:               "../../shared/pages/group.404/group.404.gitlab-example.com/public",
					IsNamespaceProject: true,
					HasAccessControl:   false,
				},
				subpath: "/",
			},
			expectedResponse: "Custom 404 group page",
		},
		{
			name:   "private_namespace_domain",
			domain: "group.404.gitlab-example.com",
			path:   "/unknown",
			resolver: &stubbedResolver{
				project: &serving.LookupPath{
					Path:               "../../shared/pages/group.404/group.404.gitlab-example.com/public",
					IsNamespaceProject: true,
					HasAccessControl:   true,
				},
				subpath: "/",
			},
			expectedResponse: "The page you're looking for could not be found.",
		},
		{
			name:   "no_parent_namespace_domain",
			domain: "group.404.gitlab-example.com",
			path:   "/unknown",
			resolver: &stubbedResolver{
				project: nil,
				subpath: "/",
			},
			expectedResponse: "The page you're looking for could not be found.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Domain{
				Name:     tt.domain,
				Resolver: tt.resolver,
			}
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", fmt.Sprintf("http://%s%s", tt.domain, tt.path), nil)
			d.serveNamespaceNotFound(w, r)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotFound, resp.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedResponse)
		})
	}
}
