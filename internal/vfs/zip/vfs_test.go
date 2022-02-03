package zip

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

func TestVFSRoot(t *testing.T) {
	u, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	tests := map[string]struct {
		path        string
		sha256      string
		expectedErr error
	}{
		"zip_file_exists": {
			path:   "/public.zip",
			sha256: "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
		},
		"zip_file_does_not_exist": {
			path:        "/unknown",
			sha256:      "filedoesnotexist",
			expectedErr: fs.ErrNotExist,
		},
		"invalid_url": {
			path:        "/%",
			sha256:      "invalidurl",
			expectedErr: url.EscapeError("%"),
		},
	}

	vfs := New(&zipCfg)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			root, err := vfs.Root(context.Background(), u+tt.path, tt.sha256)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.IsType(t, &zipArchive{}, root)

			f, err := root.Open(context.Background(), "index.html")
			require.NoError(t, err)

			content, err := io.ReadAll(f)
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

	vfs := New(&zipCfg).(*zipVFS)
	key := "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75"
	root, err := vfs.Root(context.Background(), path, key)
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
				vfs.cache.SetDefault(key, root)
			}
		}
	}()

	require.Eventually(t, func() bool {
		_, err := vfs.findOrOpenArchive(context.Background(), key, path)
		return errors.Is(err, errAlreadyCached)
	}, 3*time.Second, time.Nanosecond)
}

func TestVFSFindOrOpenArchiveRefresh(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	// It should be large enough to not have flaky executions
	const expiryInterval = 10 * time.Millisecond

	tests := map[string]struct {
		path               string
		sha256             string
		expirationInterval time.Duration
		refreshInterval    time.Duration

		expectNewArchive       bool
		expectOpenError        bool
		expectArchiveRefreshed bool
	}{
		"after cache expiry of successful open a new archive is returned": {
			path:               "/public.zip",
			sha256:             "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
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
			sha256:                 "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			expirationInterval:     time.Second,
			refreshInterval:        time.Second, // refresh always
			expectNewArchive:       false,
			expectOpenError:        false,
			expectArchiveRefreshed: true,
		},
		"subsequent open before refresh interval does not refresh archive": {
			path:                   "/public.zip",
			sha256:                 "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
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
				cfg := zipCfg
				cfg.ExpirationInterval = test.expirationInterval
				cfg.RefreshInterval = test.refreshInterval

				vfs := New(&cfg).(*zipVFS)

				path := testServerURL + test.path

				// create a new archive and increase counters
				archive1, err1 := vfs.findOrOpenArchive(context.Background(), test.sha256, path)
				if test.expectOpenError {
					require.Error(t, err1)
					require.Nil(t, archive1)
				} else {
					require.NoError(t, err1)
				}

				item1, exp1, found := vfs.cache.GetWithExpiration(test.sha256)
				require.True(t, found)

				// give some time to for timeouts to fire
				time.Sleep(expiryInterval)

				if test.expectNewArchive {
					// should return a new archive
					archive2, err2 := vfs.findOrOpenArchive(context.Background(), test.sha256, path)
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
				archive2, err2 := vfs.findOrOpenArchive(context.Background(), test.sha256, path)
				require.Equal(t, archive1, archive2, "same archive is returned")
				require.Equal(t, err1, err2, "same error for the same archive")

				item2, exp2, found := vfs.cache.GetWithExpiration(test.sha256)
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

func TestVFSReconfigureTransport(t *testing.T) {
	chdir := false
	cleanup := testhelpers.ChdirInPath(t, "../../../shared/pages", &chdir)
	defer cleanup()

	fileURL := testhelpers.ToFileProtocol(t, "group/zip.gitlab.io/public.zip")

	vfs := New(&zipCfg)
	key := "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75"

	// try to open a file URL without registering the file protocol
	_, err := vfs.Root(context.Background(), fileURL, key)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported protocol scheme \"file\"")

	// reconfigure VFS with allowed paths and try to open file://
	cfg := zipCfg
	cfg.AllowedPaths = []string{testhelpers.Getwd(t)}

	err = vfs.Reconfigure(&config.Config{Zip: cfg})
	require.NoError(t, err)

	root, err := vfs.Root(context.Background(), fileURL, key)
	require.NoError(t, err)

	fi, err := root.Lstat(context.Background(), "index.html")
	require.NoError(t, err)
	require.Equal(t, "index.html", fi.Name())
}

func withExpectedArchiveCount(t *testing.T, archiveCount int, fn func(t *testing.T)) {
	t.Helper()

	archivesMetric := metrics.ZipCachedEntries.WithLabelValues("archive")
	archivesCount := testutil.ToFloat64(archivesMetric)

	fn(t)

	archivesCountEnd := testutil.ToFloat64(archivesMetric)
	require.Equal(t, float64(archiveCount), archivesCountEnd-archivesCount, "exact number of archives is cached")
}
