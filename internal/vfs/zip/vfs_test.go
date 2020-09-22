package zip

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVFSRoot(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip")
	defer cleanup()

	tests := map[string]struct {
		path           string
		expectedErrMsg string
	}{
		"zip_file_exists": {
			path: "/public.zip",
		},
		"zip_file_does_not_exist": {
			path:           "/unknown",
			expectedErrMsg: "404 Not Found",
		},
		"invalid_url": {
			path:           "/%",
			expectedErrMsg: "invalid URL",
		},
	}

	testZipVFS := New("zip_test")

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rootVFS, err := testZipVFS.Root(context.Background(), testServerURL+tt.path)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.IsType(t, &zipArchive{}, rootVFS)

			f, err := rootVFS.Open(context.Background(), "index.html")
			require.NoError(t, err)

			content, err := ioutil.ReadAll(f)
			require.NoError(t, err)
			require.Equal(t, "zip.gitlab.io/project/index.html\n", string(content))

			fi, err := rootVFS.Lstat(context.Background(), "index.html")
			require.NoError(t, err)
			require.Equal(t, "index.html", fi.Name())

			link, err := rootVFS.Readlink(context.Background(), "symlink.html")
			require.NoError(t, err)
			require.Equal(t, "subdir/linked.html", link)
		})
	}
}

func TestVFSRootMultipleRequests(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip")
	defer cleanup()

	testZipVFS := New("zip_test")

	wg := &sync.WaitGroup{}
	wg.Add(5)

	for i := 0; i < 5; i++ {
		go func(i int) {
			defer wg.Done()

			vfs, err := testZipVFS.Root(context.Background(), testServerURL+"/public.zip")
			require.NoError(t, err, i)

			f, err := vfs.Open(context.Background(), "index.html")
			require.NoError(t, err, i)

			content, err := ioutil.ReadAll(f)
			require.NoError(t, err, i)

			require.Equal(t, "zip.gitlab.io/project/index.html\n", string(content), i)
		}(i)
	}

	wg.Wait()
	// TODO: add tests for cache callbacks https://gitlab.com/gitlab-org/gitlab-pages/-/issues/465
}
