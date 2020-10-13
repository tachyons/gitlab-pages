package zip

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

var chdirSet = false

func TestOpen(t *testing.T) {
	zip, cleanup := openZipArchive(t, nil)
	defer cleanup()

	tests := map[string]struct {
		file            string
		expectedContent string
		expectedErr     error
	}{
		"file_exists": {
			file:            "index.html",
			expectedContent: "zip.gitlab.io/project/index.html\n",
			expectedErr:     nil,
		},
		"file_exists_in_subdir": {
			file:            "subdir/hello.html",
			expectedContent: "zip.gitlab.io/project/subdir/hello.html\n",
			expectedErr:     nil,
		},
		"file_exists_symlink": {
			file:            "symlink.html",
			expectedContent: "subdir/linked.html",
			expectedErr:     errNotFile,
		},
		"is_dir": {
			file:        "subdir",
			expectedErr: errNotFile,
		},
		"file_does_not_exist": {
			file:        "unknown.html",
			expectedErr: os.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := zip.Open(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			data, err := ioutil.ReadAll(f)
			require.NoError(t, err)

			require.Equal(t, tt.expectedContent, string(data))
			require.NoError(t, f.Close())
		})
	}
}

func TestOpenCached(t *testing.T) {
	var requests int64
	zip, cleanup := openZipArchive(t, &requests)
	defer cleanup()

	t.Run("open file first time", func(t *testing.T) {
		requestsStart := requests
		f, err := zip.Open(context.Background(), "index.html")
		require.NoError(t, err)
		defer f.Close()

		_, err = ioutil.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, int64(2), atomic.LoadInt64(&requests)-requestsStart, "we expect two requests to read file: data offset and content")
	})

	t.Run("open file second time", func(t *testing.T) {
		requestsStart := atomic.LoadInt64(&requests)
		f, err := zip.Open(context.Background(), "index.html")
		require.NoError(t, err)
		defer f.Close()

		_, err = ioutil.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, int64(1), atomic.LoadInt64(&requests)-requestsStart, "we expect one request to read file with cached data offset")
	})
}

func TestLstat(t *testing.T) {
	zip, cleanup := openZipArchive(t, nil)
	defer cleanup()

	tests := map[string]struct {
		file         string
		isDir        bool
		isSymlink    bool
		expectedName string
		expectedErr  error
	}{
		"file_exists": {
			file:         "index.html",
			expectedName: "index.html",
		},
		"file_exists_in_subdir": {
			file:         "subdir/hello.html",
			expectedName: "hello.html",
		},
		"file_exists_symlink": {
			file:         "symlink.html",
			isSymlink:    true,
			expectedName: "symlink.html",
		},
		"has_root": {
			file:         "",
			isDir:        true,
			expectedName: "public",
		},
		"has_root_dot": {
			file:         ".",
			isDir:        true,
			expectedName: "public",
		},
		"has_root_slash": {
			file:         "/",
			isDir:        true,
			expectedName: "public",
		},
		"is_dir": {
			file:         "subdir",
			isDir:        true,
			expectedName: "subdir",
		},
		"file_does_not_exist": {
			file:        "unknown.html",
			expectedErr: os.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fi, err := zip.Lstat(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedName, fi.Name())
			require.Equal(t, tt.isDir, fi.IsDir())
			require.NotEmpty(t, fi.ModTime())

			if tt.isDir {
				require.Zero(t, fi.Size())
				require.True(t, fi.IsDir())
				return
			}

			require.NotZero(t, fi.Size())

			if tt.isSymlink {
				require.NotZero(t, fi.Mode()&os.ModeSymlink)
			} else {
				require.True(t, fi.Mode().IsRegular())
			}
		})
	}
}

func TestReadLink(t *testing.T) {
	zip, cleanup := openZipArchive(t, nil)
	defer cleanup()

	tests := map[string]struct {
		file        string
		expectedErr error
	}{
		"symlink_success": {
			file: "symlink.html",
		},
		"file": {
			file:        "index.html",
			expectedErr: errNotSymlink,
		},
		"dir": {
			file:        "subdir",
			expectedErr: errNotSymlink,
		},
		"symlink_too_big": {
			file:        "bad_symlink.html",
			expectedErr: errSymlinkSize,
		},
		"file_does_not_exist": {
			file:        "unknown.html",
			expectedErr: os.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			link, err := zip.Readlink(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, link)
		})
	}
}

func TestReadlinkCached(t *testing.T) {
	var requests int64
	zip, cleanup := openZipArchive(t, &requests)
	defer cleanup()

	t.Run("readlink first time", func(t *testing.T) {
		requestsStart := atomic.LoadInt64(&requests)
		_, err := zip.Readlink(context.Background(), "symlink.html")
		require.NoError(t, err)
		require.Equal(t, int64(2), atomic.LoadInt64(&requests)-requestsStart, "we expect two requests to read symlink: data offset and link")
	})

	t.Run("readlink second time", func(t *testing.T) {
		requestsStart := atomic.LoadInt64(&requests)
		_, err := zip.Readlink(context.Background(), "symlink.html")
		require.NoError(t, err)
		require.Equal(t, int64(0), atomic.LoadInt64(&requests)-requestsStart, "we expect no additional requests to read cached symlink")
	})
}

func TestArchiveCanBeReadAfterOpenCtxCanceled(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	fs := New().(*zipVFS)
	zip := newArchive(fs, testServerURL+"/public.zip", time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := zip.openArchive(ctx)
	require.EqualError(t, err, context.Canceled.Error())

	<-zip.done

	file, err := zip.Open(context.Background(), "index.html")
	require.NoError(t, err)
	data, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	require.Equal(t, "zip.gitlab.io/project/index.html\n", string(data))
	require.NoError(t, file.Close())
}

func TestReadArchiveFails(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	fs := New().(*zipVFS)
	zip := newArchive(fs, testServerURL+"/unkown.html", time.Second)

	err := zip.openArchive(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Not Found")

	_, err = zip.Open(context.Background(), "index.html")
	require.EqualError(t, err, os.ErrNotExist.Error())
}

func openZipArchive(t *testing.T, requests *int64) (*zipArchive, func()) {
	t.Helper()

	if requests == nil {
		requests = new(int64)
	}

	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public-without-dirs.zip", requests)

	fs := New().(*zipVFS)
	zip := newArchive(fs, testServerURL+"/public.zip", time.Second)

	err := zip.openArchive(context.Background())
	require.NoError(t, err)

	// public/ public/index.html public/404.html public/symlink.html
	// public/subdir/ public/subdir/hello.html public/subdir/linked.html
	// public/bad_symlink.html public/subdir/2bp3Qzs...
	require.NotZero(t, zip.files)
	require.Equal(t, int64(3), atomic.LoadInt64(requests), "we expect three requests to open ZIP archive: size and two to seek central directory")

	return zip, func() {
		cleanup()
	}
}

func newZipFileServerURL(t *testing.T, zipFilePath string, requests *int64) (string, func()) {
	t.Helper()

	chdir := testhelpers.ChdirInPath(t, "../../../shared/pages", &chdirSet)

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
		if requests != nil {
			atomic.AddInt64(requests, 1)
		}
	}))

	testServer := httptest.NewServer(m)

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}
