package gitlab

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/mock"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

type lookupPathTest struct {
	file                string
	target              string
	expectedPrefix      string
	expectedPath        string
	expectedSubPath     string
	expectedIsNamespace bool
}

func TestGetDomain(t *testing.T) {
	tests := map[string]struct {
		file          string
		domain        string
		mockLookup    *api.Lookup
		expectedError error
	}{
		"when the response is correct": {
			file:   "client/testdata/test.gitlab.io.json",
			domain: "test.gitlab.io",
		},
		"when the response is not valid": {
			file:          "/dev/null",
			domain:        "test.gitlab.io",
			expectedError: io.EOF,
		},
		"when the response is unauthorized": {
			mockLookup:    &api.Lookup{Error: client.ErrUnauthorizedAPI},
			domain:        "test",
			expectedError: client.ErrUnauthorizedAPI,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := NewMockClient(t, tc.file, tc.mockLookup, false)
			source := Gitlab{client: mockClient}

			domain, err := source.GetDomain(context.Background(), tc.domain)
			if tc.expectedError == nil {
				require.NoError(t, err)
				require.Equal(t, tc.domain, domain.Name)
			} else {
				require.Error(t, err)
				require.Nil(t, domain)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := map[string]struct {
		file                string
		target              string
		expectedPrefix      string
		expectedPath        string
		expectedSubPath     string
		expectedIsNamespace bool
	}{
		"when requesting nested group project with root path": {
			file:                "client/testdata/test.gitlab.io.json",
			target:              "https://test.gitlab.io:443/my/pages/project/",
			expectedPrefix:      "/my/pages/project/",
			expectedPath:        "some/path/to/project/",
			expectedSubPath:     "",
			expectedIsNamespace: false,
		},
		"when requesting a nested group project with full path": {
			file:                "client/testdata/test.gitlab.io.json",
			target:              "https://test.gitlab.io:443/my/pages/project/path/index.html",
			expectedPrefix:      "/my/pages/project/",
			expectedPath:        "some/path/to/project/",
			expectedSubPath:     "path/index.html",
			expectedIsNamespace: false,
		},
		"when requesting the group root project with root path": {
			file:                "client/testdata/test.gitlab.io.json",
			target:              "https://test.gitlab.io:443/",
			expectedPrefix:      "/",
			expectedPath:        "some/path/to/project-3/",
			expectedSubPath:     "",
			expectedIsNamespace: true,
		},
		"when requesting the group root project with full path": {
			file:                "client/testdata/test.gitlab.io.json",
			target:              "https://test.gitlab.io:443/path/to/index.html",
			expectedPrefix:      "/",
			expectedPath:        "some/path/to/project-3/",
			expectedSubPath:     "path/to/index.html",
			expectedIsNamespace: true,
		},
		"when request path has not been sanitized": {
			file:            "client/testdata/test.gitlab.io.json",
			target:          "https://test.gitlab.io:443/something/../something/../my/pages/project/index.html",
			expectedPrefix:  "/my/pages/project/",
			expectedPath:    "some/path/to/project/",
			expectedSubPath: "index.html",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := NewMockClient(t, tc.file, nil, false)
			source := Gitlab{client: mockClient, enableDisk: true}

			request := httptest.NewRequest(http.MethodGet, tc.target, nil)

			response, err := source.Resolve(request)
			require.NoError(t, err)

			require.Equal(t, tc.expectedPrefix, response.LookupPath.Prefix)
			require.Equal(t, tc.expectedPath, response.LookupPath.Path)
			require.Equal(t, tc.expectedSubPath, response.SubPath)
			require.Equal(t, tc.expectedIsNamespace, response.LookupPath.IsNamespaceProject)
		})
	}
}

// Test validates https://gitlab.com/gitlab-org/gitlab-pages/-/issues/646 with go test -race
func TestResolveLookupPathsConcurrentNetRequests(t *testing.T) {
	tests := map[string]lookupPathTest{
		"when requesting the group root project with root path": {
			file:                "client/testdata/group-first.gitlab.io.json",
			target:              "https://group-first.gitlab.io:443/",
			expectedPrefix:      "/",
			expectedPath:        "some/path/group/",
			expectedSubPath:     "",
			expectedIsNamespace: true,
		},
		"when requesting another project with path": {
			file:                "client/testdata/group-first.gitlab.io.json",
			target:              "https://group-first.gitlab.io:443/my/second-project/index.html",
			expectedPrefix:      "/my/second-project/",
			expectedPath:        "some/path/to/project-2/",
			expectedSubPath:     "index.html",
			expectedIsNamespace: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockClient := NewMockClient(t, test.file, nil, true)
			cache := cache.NewCache(mockClient, &testhelpers.CacheConfig)

			for i := 0; i < 3; i++ {
				wg.Add(1)
				go sendResolveRequest(t, wg, cache, test)
			}

			wg.Wait()
		})
	}
}

// Test proves fix for https://gitlab.com/gitlab-org/gitlab-pages/-/issues/576
func TestResolveLookupPathsOrderDoesNotMatter(t *testing.T) {
	tests := map[string]lookupPathTest{
		"when requesting the group root project with root path": {
			file:                "client/testdata/group-first.gitlab.io.json",
			target:              "https://group-first.gitlab.io:443/",
			expectedPrefix:      "/",
			expectedPath:        "some/path/group/",
			expectedSubPath:     "",
			expectedIsNamespace: true,
		},
		"when requesting another project with path": {
			file:                "client/testdata/group-first.gitlab.io.json",
			target:              "https://group-first.gitlab.io:443/my/second-project/index.html",
			expectedPrefix:      "/my/second-project/",
			expectedPath:        "some/path/to/project-2/",
			expectedSubPath:     "index.html",
			expectedIsNamespace: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := NewMockClient(t, test.file, nil, true)
			cache := cache.NewCache(mockClient, &testhelpers.CacheConfig)
			source := Gitlab{client: cache, enableDisk: true}

			request := httptest.NewRequest(http.MethodGet, test.target, nil)

			response, err := source.Resolve(request)
			require.NoError(t, err)

			require.Equal(t, test.expectedPrefix, response.LookupPath.Prefix)
			require.Equal(t, test.expectedPath, response.LookupPath.Path)
			require.Equal(t, test.expectedSubPath, response.SubPath)
			require.Equal(t, test.expectedIsNamespace, response.LookupPath.IsNamespaceProject)
		})
	}
}

func NewMockClient(t *testing.T, file string, mockedLookup *api.Lookup, useCache bool) *mock.MockClientStub {
	mockCtrl := gomock.NewController(t)

	mockClient := mock.NewMockClientStub(mockCtrl)
	if !useCache {
		mockClient.EXPECT().
			Resolve(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, domain string) *api.Lookup {
				lookup := mockClient.GetLookup(ctx, domain)
				return &lookup
			}).
			Times(1)
	}

	mockClient.EXPECT().
		GetLookup(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, domain string) api.Lookup {
			if mockedLookup != nil {
				return *mockedLookup
			}

			lookup := api.Lookup{Name: domain}

			f, err := os.Open(file)
			if err != nil {
				lookup.Error = err
				return lookup
			}
			defer f.Close()

			lookup.ParseDomain(f)

			return lookup
		}).
		Times(1)

	return mockClient
}

func sendResolveRequest(t *testing.T, wg *sync.WaitGroup, resolver api.Resolver, test lookupPathTest) {
	request := httptest.NewRequest(http.MethodGet, test.target, nil)

	source := Gitlab{client: resolver, enableDisk: true}

	response, err := source.Resolve(request)
	require.NoError(t, err)

	require.Equal(t, test.expectedPrefix, response.LookupPath.Prefix)
	require.Equal(t, test.expectedPath, response.LookupPath.Path)
	require.Equal(t, test.expectedSubPath, response.SubPath)
	require.Equal(t, test.expectedIsNamespace, response.LookupPath.IsNamespaceProject)

	wg.Done()
}
