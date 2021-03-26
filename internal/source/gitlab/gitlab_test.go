package gitlab

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

func TestGetDomain(t *testing.T) {
	t.Run("when the response if correct", func(t *testing.T) {
		client := client.StubClient{File: "client/testdata/test.gitlab.io.json"}
		source := Gitlab{client: client}

		domain, err := source.GetDomain("test.gitlab.io")
		require.NoError(t, err)

		require.Equal(t, "test.gitlab.io", domain.Name)
	})

	t.Run("when the response is not valid", func(t *testing.T) {
		client := client.StubClient{File: "/dev/null"}
		source := Gitlab{client: client}

		domain, err := source.GetDomain("test.gitlab.io")

		require.NotNil(t, err)
		require.Nil(t, domain)
	})

	t.Run("when pages endpoint is unauthorized", func(t *testing.T) {
		c := client.StubClient{Lookup: &api.Lookup{Error: client.ErrUnauthorizedAPI}}
		source := Gitlab{client: c}

		_, err := source.GetDomain("test")
		require.EqualError(t, err, client.ErrUnauthorizedAPI.Error())
		require.False(t, source.IsReady())
	})
}

func TestResolve(t *testing.T) {
	client := client.StubClient{File: "client/testdata/test.gitlab.io.json"}
	source := Gitlab{client: client}

	t.Run("when requesting nested group project with root path", func(t *testing.T) {
		target := "https://test.gitlab.io:443/my/pages/project/"
		request := httptest.NewRequest("GET", target, nil)

		response, err := source.Resolve(request)
		require.NoError(t, err)

		require.Equal(t, "/my/pages/project/", response.LookupPath.Prefix)
		require.Equal(t, "some/path/to/project/", response.LookupPath.Path)
		require.Equal(t, "", response.SubPath)
		require.False(t, response.LookupPath.IsNamespaceProject)
	})

	t.Run("when requesting a nested group project with full path", func(t *testing.T) {
		target := "https://test.gitlab.io:443/my/pages/project/path/index.html"
		request := httptest.NewRequest("GET", target, nil)

		response, err := source.Resolve(request)
		require.NoError(t, err)

		require.Equal(t, "/my/pages/project/", response.LookupPath.Prefix)
		require.Equal(t, "some/path/to/project/", response.LookupPath.Path)
		require.Equal(t, "path/index.html", response.SubPath)
		require.False(t, response.LookupPath.IsNamespaceProject)
	})

	t.Run("when requesting the group root project with root path", func(t *testing.T) {
		target := "https://test.gitlab.io:443/"
		request := httptest.NewRequest("GET", target, nil)

		response, err := source.Resolve(request)
		require.NoError(t, err)

		require.Equal(t, "/", response.LookupPath.Prefix)
		require.Equal(t, "some/path/to/project-3/", response.LookupPath.Path)
		require.Equal(t, "", response.SubPath)
		require.True(t, response.LookupPath.IsNamespaceProject)
	})

	t.Run("when requesting the group root project with full path", func(t *testing.T) {
		target := "https://test.gitlab.io:443/path/to/index.html"
		request := httptest.NewRequest("GET", target, nil)

		response, err := source.Resolve(request)
		require.NoError(t, err)

		require.Equal(t, "/", response.LookupPath.Prefix)
		require.Equal(t, "path/to/index.html", response.SubPath)
		require.Equal(t, "some/path/to/project-3/", response.LookupPath.Path)
		require.True(t, response.LookupPath.IsNamespaceProject)
	})

	t.Run("when request path has not been sanitized", func(t *testing.T) {
		target := "https://test.gitlab.io:443/something/../something/../my/pages/project/index.html"
		request := httptest.NewRequest("GET", target, nil)

		response, err := source.Resolve(request)
		require.NoError(t, err)

		require.Equal(t, "/my/pages/project/", response.LookupPath.Prefix)
		require.Equal(t, "index.html", response.SubPath)
	})
}
