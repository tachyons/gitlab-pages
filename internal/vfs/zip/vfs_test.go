package zip

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

func TestVFSRoot(t *testing.T) {
	url, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
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
			expectedErrMsg: vfs.ErrNotExist{Inner: httprange.ErrNotFound}.Error(),
		},
		"invalid_url": {
			path:           "/%",
			expectedErrMsg: "invalid URL",
		},
	}

	vfs := New()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			root, err := vfs.Root(context.Background(), url+tt.path)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.IsType(t, &zipArchive{}, root)

			f, err := root.Open(context.Background(), "index.html")
			require.NoError(t, err)

			content, err := ioutil.ReadAll(f)
			require.NoError(t, err)
			require.Equal(t, "zip.gitlab.io/project/index.html\n", string(content))

			fi, err := root.Lstat(context.Background(), "index.html")
			require.NoError(t, err)
			require.Equal(t, "index.html", fi.Name())

			link, err := root.Readlink(context.Background(), "symlink.html")
			require.NoError(t, err)
			require.Equal(t, "subdir/linked.html", link)
		})
	}
}

func TestVFSFindOrOpenArchiveConcurrentAccess(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	path := testServerURL + "/public.zip"

	vfs := New().(*zipVFS)
	root, err := vfs.Root(context.Background(), path)
	require.NoError(t, err)

	done := make(chan struct{})
	defer close(done)

	// Try to hit a condition between the invocation
	// of cache.GetWithExpiration and cache.Add
	go func() {
		for {
			select {
			case <-done:
				return

			default:
				vfs.cache.Flush()
				vfs.cache.SetDefault(path, root)
			}
		}
	}()

	require.Eventually(t, func() bool {
		_, err := vfs.findOrOpenArchive(context.Background(), path)
		return err == errAlreadyCached
	}, time.Second, time.Nanosecond)
}

func TestVFSArchiveCacheEvict(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	path := testServerURL + "/public.zip"

	vfs := New(
		WithCacheExpirationInterval(time.Nanosecond),
	).(*zipVFS)

	archivesMetric := metrics.ZipCachedEntries.WithLabelValues("archive")
	archivesCount := testutil.ToFloat64(archivesMetric)

	// create a new archive and increase counters
	archive, err := vfs.Root(context.Background(), path)
	require.NoError(t, err)
	require.NotNil(t, archive)

	// wait for archive to expire
	time.Sleep(time.Nanosecond)

	// a new object is created
	archive2, err := vfs.Root(context.Background(), path)
	require.NoError(t, err)
	require.NotNil(t, archive2)
	require.NotEqual(t, archive, archive2, "a different archive is returned")

	archivesCountEnd := testutil.ToFloat64(archivesMetric)
	require.Equal(t, float64(1), archivesCountEnd-archivesCount, "all expired archives are evicted")
}

func TestVFSFindOrCreateArchiveErrored(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	path := testServerURL + "/unknown.zip"

	vfs := New().(*zipVFS)

	archivesMetric := metrics.ZipCachedEntries.WithLabelValues("archive")
	archivesCount := testutil.ToFloat64(archivesMetric)

	// create a new archive and increase counters
	archive, err := vfs.findOrOpenArchive(context.Background(), path)
	require.Error(t, err)
	require.Nil(t, archive)

	item, exp1, found := vfs.cache.GetWithExpiration(path)
	require.True(t, found)

	// item has been opened and has an error
	opened, err1 := item.(*zipArchive).openStatus()
	require.True(t, opened)
	require.Error(t, err1)
	require.Equal(t, err, err1, "same error as findOrOpenArchive")

	// should return errored archive
	archive2, err2 := vfs.findOrOpenArchive(context.Background(), path)
	require.Error(t, err2)
	require.Nil(t, archive2)
	require.Equal(t, err1, err2, "same error for the same archive")

	_, exp2, found := vfs.cache.GetWithExpiration(path)
	require.True(t, found)
	require.Equal(t, exp1, exp2, "archive has not been refreshed")

	archivesCountEnd := testutil.ToFloat64(archivesMetric)
	require.Equal(t, float64(1), archivesCountEnd-archivesCount, "only one archive cached")
}
