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
		_, err := vfs.findOrOpenArchive(context.Background(), path, path)
		return err == errAlreadyCached
	}, time.Second, time.Nanosecond)
}

func TestVFSFindOrOpenArchiveRefresh(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	// It should be large enough to not have flaky executions
	const expiryInterval = 10 * time.Millisecond

	tests := map[string]struct {
		path               string
		expirationInterval time.Duration
		refreshInterval    time.Duration

		expectNewArchive       bool
		expectOpenError        bool
		expectArchiveRefreshed bool
	}{
		"after cache expiry of successful open a new archive is returned": {
			path:               "/public.zip",
			expirationInterval: expiryInterval,
			expectNewArchive:   true,
			expectOpenError:    false,
		},
		"after cache expiry of errored open a new archive is returned": {
			path:               "/unknown.zip",
			expirationInterval: expiryInterval,
			expectNewArchive:   true,
			expectOpenError:    true,
		},
		"subsequent open during refresh interval does refresh archive": {
			path:                   "/public.zip",
			expirationInterval:     time.Second,
			refreshInterval:        time.Second, // refresh always
			expectNewArchive:       false,
			expectOpenError:        false,
			expectArchiveRefreshed: true,
		},
		"subsequent open before refresh interval does not refresh archive": {
			path:                   "/public.zip",
			expirationInterval:     time.Second,
			refreshInterval:        time.Millisecond, // very short interval should not refresh
			expectNewArchive:       false,
			expectOpenError:        false,
			expectArchiveRefreshed: false,
		},
		"subsequent open of errored archive during refresh interval does not refresh": {
			path:                   "/unknown.zip",
			expirationInterval:     time.Second,
			refreshInterval:        time.Second, // refresh always (if not error)
			expectNewArchive:       false,
			expectOpenError:        true,
			expectArchiveRefreshed: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			withExpectedArchiveCount(t, 1, func(t *testing.T) {
				vfs := New(
					WithCacheExpirationInterval(test.expirationInterval),
					WithCacheRefreshInterval(test.refreshInterval),
				).(*zipVFS)

				path := testServerURL + test.path

				// create a new archive and increase counters
				archive1, err1 := vfs.findOrOpenArchive(context.Background(), path, path)
				if test.expectOpenError {
					require.Error(t, err1)
					require.Nil(t, archive1)
				} else {
					require.NoError(t, err1)
				}

				item1, exp1, found := vfs.cache.GetWithExpiration(path)
				require.True(t, found)

				// give some time to for timeouts to fire
				time.Sleep(expiryInterval)

				if test.expectNewArchive {
					// should return a new archive
					archive2, err2 := vfs.findOrOpenArchive(context.Background(), path, path)
					if test.expectOpenError {
						require.Error(t, err2)
						require.Nil(t, archive2)
					} else {
						require.NoError(t, err2)
						require.NotEqual(t, archive1, archive2, "a new archive should be returned")
					}
					return
				}

				// should return exactly the same archive
				archive2, err2 := vfs.findOrOpenArchive(context.Background(), path, path)
				require.Equal(t, archive1, archive2, "same archive is returned")
				require.Equal(t, err1, err2, "same error for the same archive")

				item2, exp2, found := vfs.cache.GetWithExpiration(path)
				require.True(t, found)
				require.Equal(t, item1, item2, "same item is returned")

				if test.expectArchiveRefreshed {
					require.Greater(t, exp2.UnixNano(), exp1.UnixNano(), "archive should be refreshed")
				} else {
					require.Equal(t, exp1.UnixNano(), exp2.UnixNano(), "archive has not been refreshed")
				}
			})
		})
	}
}

func withExpectedArchiveCount(t *testing.T, archiveCount int, fn func(t *testing.T)) {
	t.Helper()

	archivesMetric := metrics.ZipCachedEntries.WithLabelValues("archive")
	archivesCount := testutil.ToFloat64(archivesMetric)

	fn(t)

	archivesCountEnd := testutil.ToFloat64(archivesMetric)
	require.Equal(t, float64(archiveCount), archivesCountEnd-archivesCount, "exact number of archives is cached")
}
