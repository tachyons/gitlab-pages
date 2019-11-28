package gitlab

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

func TestGetDomain(t *testing.T) {
	client := client.NewStubClient("client/testdata/test.gitlab.io.json")
	source := Gitlab{client: client, cache: cache.New()}

	domain, err := source.GetDomain("test.gitlab.io")
	require.NoError(t, err)

	assert.Equal(t, "test.gitlab.io", domain.Name)
}

func TestResolve(t *testing.T) {
	client := client.NewStubClient("client/testdata/test.gitlab.io.json")
	source := Gitlab{client: client, cache: cache.New()}

	t.Run("when requesting a nested group project", func(t *testing.T) {
		target := "https://test.gitlab.io:443/my/pages/project/path/index.html"
		request := httptest.NewRequest("GET", target, nil)

		lookup, subpath, err := source.Resolve(request)
		require.NoError(t, err)

		assert.Equal(t, "/path/index.html", subpath)
		assert.Equal(t, "/my/pages/project", lookup.Location)
		assert.False(t, lookup.IsNamespaceProject)
	})

	t.Run("when request a nested group project", func(t *testing.T) {
		target := "https://test.gitlab.io:443/path/to/index.html"
		request := httptest.NewRequest("GET", target, nil)

		lookup, subpath, err := source.Resolve(request)
		require.NoError(t, err)

		assert.Equal(t, "path/to/index.html", subpath)
		assert.Equal(t, "some/path/to/project-3/", lookup.Path)
		assert.Equal(t, "/", lookup.Location)
		assert.True(t, lookup.IsNamespaceProject)
	})
}
