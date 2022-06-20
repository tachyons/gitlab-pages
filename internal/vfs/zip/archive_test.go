package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var (
	chdirSet = false
	zipCfg   = config.ZipServing{
		ExpirationInterval: 10 * time.Second,
		CleanupInterval:    5 * time.Second,
		RefreshInterval:    5 * time.Second,
		OpenTimeout:        5 * time.Second,
	}
)

func TestOpen(t *testing.T) {
	t.Run("open_from_server", runZipTest(t, testOpen, false))
	t.Run("open_from_disk", runZipTest(t, testOpen, true))
}

func testOpen(t *testing.T, zip *zipArchive) {
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
			expectedErr: fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := zip.Open(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			data, err := io.ReadAll(f)
			require.NoError(t, err)

			require.Equal(t, tt.expectedContent, string(data))
			require.NoError(t, f.Close())
		})
	}
}

func TestOpenCached(t *testing.T) {
	var requests int64
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public-without-dirs.zip", &requests)
	defer cleanup()

	fs := New(&zipCfg)

	// We use array instead of map to ensure
	// predictable ordering of test execution
	tests := []struct {
		name                  string
		vfsPath               string
		sha256                string
		filePath              string
		expectedArchiveStatus archiveStatus
		expectedOpenErr       error
		expectedReadErr       error
		expectedRequests      int64
	}{
		{
			name:     "open file first time",
			vfsPath:  testServerURL + "/public.zip",
			sha256:   "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath: "index.html",
			// we expect five requests to:
			// read resource and zip metadata
			// read file: data offset and content
			expectedRequests:      5,
			expectedArchiveStatus: archiveOpened,
		},
		{
			name:     "open file second time",
			vfsPath:  testServerURL + "/public.zip",
			sha256:   "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath: "index.html",
			// we expect one request to read file with cached data offset
			expectedRequests:      1,
			expectedArchiveStatus: archiveOpened,
		},
		{
			name:                  "when the URL changes",
			vfsPath:               testServerURL + "/public.zip?new-secret",
			sha256:                "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath:              "index.html",
			expectedRequests:      1,
			expectedArchiveStatus: archiveOpened,
		},
		{
			name:             "when opening cached file and content changes",
			vfsPath:          testServerURL + "/public.zip?changed-content=1",
			sha256:           "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath:         "index.html",
			expectedRequests: 1,
			// we receive an error on `read` as `open` offset is already cached
			expectedReadErr:       vfs.NewReadError(httprange.ErrRangeRequestsNotSupported),
			expectedArchiveStatus: archiveCorrupted,
		},
		{
			name:                  "after content change archive is reloaded",
			vfsPath:               testServerURL + "/public.zip?new-secret",
			sha256:                "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath:              "index.html",
			expectedRequests:      5,
			expectedArchiveStatus: archiveOpened,
		},
		{
			name:             "when opening non-cached file and content changes",
			vfsPath:          testServerURL + "/public.zip?changed-content=1",
			sha256:           "d6b318b399cfe9a1c8483e49847ee49a2676d8cfd6df57ec64d971ad03640a75",
			filePath:         "subdir/hello.html",
			expectedRequests: 1,
			// we receive an error on `read` as `open` offset is already cached
			expectedOpenErr:       vfs.NewReadError(httprange.ErrRangeRequestsNotSupported),
			expectedArchiveStatus: archiveCorrupted,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := atomic.LoadInt64(&requests)
			zip, err := fs.Root(context.Background(), test.vfsPath, test.sha256)
			require.NoError(t, err)

			f, err := zip.Open(context.Background(), test.filePath)
			if test.expectedOpenErr != nil {
				require.Equal(t, test.expectedOpenErr, err)
				status, _ := zip.(*zipArchive).openStatus()
				require.Equal(t, test.expectedArchiveStatus, status)
				return
			}

			require.NoError(t, err)
			defer f.Close()

			_, err = io.ReadAll(f)
			if test.expectedReadErr != nil {
				require.Equal(t, test.expectedReadErr, err)
				status, _ := zip.(*zipArchive).openStatus()
				require.Equal(t, test.expectedArchiveStatus, status)
				return
			}

			require.NoError(t, err)
			status, _ := zip.(*zipArchive).openStatus()
			require.Equal(t, test.expectedArchiveStatus, status)

			end := atomic.LoadInt64(&requests)
			require.Equal(t, test.expectedRequests, end-start)
		})
	}
}

func TestLstat(t *testing.T) {
	t.Run("lstat_from_server", runZipTest(t, testLstat, false))
	t.Run("lstat_from_disk", runZipTest(t, testLstat, true))
}

func testLstat(t *testing.T, zip *zipArchive) {
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
			expectedErr: fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fi, err := zip.Lstat(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
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
	t.Run("read_link_from_server", runZipTest(t, testReadLink, false))
	t.Run("read_link_from_disk", runZipTest(t, testReadLink, true))
}

func testReadLink(t *testing.T, zip *zipArchive) {
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
			expectedErr: fs.ErrNotExist,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			link, err := zip.Readlink(context.Background(), tt.file)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, link)
		})
	}
}

func TestReadlinkCached(t *testing.T) {
	var requests int64
	zip, cleanup := openZipArchive(t, &requests, false)
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

	fs := New(&zipCfg).(*zipVFS)
	zip := newArchive(fs, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := zip.openArchive(ctx, testServerURL+"/public.zip")
	require.ErrorIs(t, err, context.Canceled)

	<-zip.done

	file, err := zip.Open(context.Background(), "index.html")
	require.NoError(t, err)
	data, err := io.ReadAll(file)
	require.NoError(t, err)

	require.Equal(t, "zip.gitlab.io/project/index.html\n", string(data))
	require.NoError(t, file.Close())
}

func TestReadArchiveFails(t *testing.T) {
	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip", nil)
	defer cleanup()

	zfs := New(&zipCfg).(*zipVFS)
	zip := newArchive(zfs, time.Second)

	err := zip.openArchive(context.Background(), testServerURL+"/unkown.html")
	require.ErrorIs(t, err, httprange.ErrNotFound)

	_, err = zip.Open(context.Background(), "index.html")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func createArchive(t *testing.T, dir string) (map[string][]byte, int64) {
	t.Helper()

	f, err := os.Create(filepath.Join(dir, "public.zip"))
	require.NoError(t, err)
	defer f.Close()
	zw := zip.NewWriter(f)

	entries := make(map[string][]byte)
	for _, size := range []int{0, 32 * 1024, 128 * 1024, 5 * 1024 * 1024} {
		entryName := fmt.Sprintf("public/file_%d", size)
		entries[entryName] = bytes.Repeat([]byte{'z'}, size)

		w, err := zw.Create(entryName)
		require.NoError(t, err)

		_, err = w.Write(entries[entryName])
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())

	fi, err := f.Stat()
	require.NoError(t, err)
	require.NoError(t, f.Close())

	return entries, fi.Size()
}

func TestMinimalRangeRequests(t *testing.T) {
	dir := t.TempDir()
	entries, size := createArchive(t, dir)

	mux := http.NewServeMux()

	var ranges []string
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rangeHdr := r.Header.Get("Range")
		if rangeHdr == "" {
			rw.Header().Add("Content-Length", fmt.Sprintf("%d", size))
			return
		}

		ranges = append(ranges, rangeHdr)

		http.FileServer(http.Dir(dir)).ServeHTTP(rw, r)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := New(&zipCfg).(*zipVFS)
	zip := newArchive(fs, time.Second)
	err := zip.openArchive(ctx, srv.URL+"/public.zip")
	require.NoError(t, err)

	require.Len(t, zip.files, len(entries))
	require.Len(t, ranges, 3, "range requests should be minimal")

	for _, zf := range zip.files {
		if !zf.Mode().IsRegular() {
			continue
		}

		f, err := zip.Open(context.Background(), strings.TrimPrefix(zf.Name, "public/"))
		require.NoError(t, err)

		io.Copy(io.Discard, f)

		require.NoError(t, f.Close())
	}

	// ensure minimal requests: https://gitlab.com/gitlab-org/gitlab-pages/-/issues/625
	require.Len(t, ranges, 11, "range requests should be minimal")
}

func openZipArchive(t *testing.T, requests *int64, fromDisk bool) (*zipArchive, func()) {
	t.Helper()

	if requests == nil {
		requests = new(int64)
	}

	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public-without-dirs.zip", requests)

	wd, err := os.Getwd()
	require.NoError(t, err)

	zipCfg.AllowedPaths = []string{wd}

	fs := New(&zipCfg).(*zipVFS)
	err = fs.Reconfigure(&config.Config{Zip: zipCfg})
	require.NoError(t, err)

	zip := newArchive(fs, time.Second)

	if fromDisk {
		fileName := testhelpers.ToFileProtocol(t, "group/zip.gitlab.io/public-without-dirs.zip")
		err := zip.openArchive(context.Background(), fileName)
		require.NoError(t, err)
	} else {
		err := zip.openArchive(context.Background(), testServerURL+"/public.zip")
		require.NoError(t, err)
		require.Equal(t, int64(3), atomic.LoadInt64(requests), "we expect three requests to open ZIP archive: size and two to seek central directory")
	}

	// public/ public/index.html public/404.html public/symlink.html
	// public/subdir/ public/subdir/hello.html public/subdir/linked.html
	// public/bad_symlink.html public/subdir/2bp3Qzs...
	require.NotZero(t, zip.files)

	return zip, func() {
		cleanup()
	}
}

func newZipFileServerURL(t *testing.T, zipFilePath string, requests *int64) (string, func()) {
	t.Helper()

	chdir := testhelpers.ChdirInPath(t, "../../../shared/pages", &chdirSet)

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests != nil {
			atomic.AddInt64(requests, 1)
		}

		r.ParseForm()

		if changedContent := r.Form.Get("changed-content"); changedContent != "" {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		http.ServeFile(w, r, zipFilePath)
	}))

	testServer := httptest.NewServer(m)

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}

func benchmarkArchiveRead(b *testing.B, size int64) {
	zbuf := new(bytes.Buffer)

	// create zip file of specified size
	zw := zip.NewWriter(zbuf)
	w, err := zw.Create("public/file.txt")
	require.NoError(b, err)
	_, err = io.CopyN(w, rand.Reader, size)
	require.NoError(b, err)
	require.NoError(b, zw.Close())

	modtime := time.Now().Add(-time.Hour)

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "public.zip", modtime, bytes.NewReader(zbuf.Bytes()))
	}))

	ts := httptest.NewServer(m)
	defer ts.Close()

	fs := New(&zipCfg).(*zipVFS)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		z := newArchive(fs, time.Second)
		err := z.openArchive(context.Background(), ts.URL+"/public.zip")
		require.NoError(b, err)

		f, err := z.Open(context.Background(), "file.txt")
		require.NoError(b, err)

		_, err = io.Copy(io.Discard, f)
		require.NoError(b, err)

		require.NoError(b, f.Close())
	}
}

func BenchmarkArchiveRead(b *testing.B) {
	for _, size := range []int{32 * 1024, 64 * 1024, 1024 * 1024} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			benchmarkArchiveRead(b, int64(size))
		})
	}
}

func runZipTest(t *testing.T, runTest func(t *testing.T, zip *zipArchive), fromDisk bool) func(t *testing.T) {
	t.Helper()

	return func(t *testing.T) {
		zip, cleanup := openZipArchive(t, nil, fromDisk)
		defer cleanup()

		runTest(t, zip)
	}
}
