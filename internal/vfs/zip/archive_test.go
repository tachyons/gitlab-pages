package zip

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

var chdirSet = false

func TestOpen(t *testing.T) {
	zip, cleanup := openZipArchive(t)
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
			expectedErr:     nil,
		},
		"is_dir": {
			file:            "subdir",
			expectedContent: "zip.gitlab.io/project/index.html\n",
			expectedErr:     nil,
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

			if name == "is_dir" {
				// cannot ioutil.ReadAll dirs but zip.Open should not fail
				return
			}

			data, err := ioutil.ReadAll(f)
			require.NoError(t, err)

			require.Equal(t, tt.expectedContent, string(data))
			require.NoError(t, f.Close())
		})
	}
}

func TestLstat(t *testing.T) {
	zip, cleanup := openZipArchive(t)
	defer cleanup()

	tests := map[string]struct {
		file        string
		isDir       bool
		isSymlink   bool
		expectedErr error
	}{
		"file_exists": {
			file: "index.html",
		},
		"file_exists_in_subdir": {
			file: "subdir/hello.html",
		},
		"file_exists_symlink": {
			file:      "symlink.html",
			isSymlink: true,
		},
		"is_dir": {
			file:  "subdir",
			isDir: true,
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
			require.Contains(t, tt.file, fi.Name())
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
	zip, cleanup := openZipArchive(t)
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

func openZipArchive(t *testing.T) (*zipArchive, func()) {
	t.Helper()

	testServerURL, cleanup := newZipFileServerURL(t, "group/zip.gitlab.io/public.zip")

	zip := newArchive(testServerURL + "/public.zip")

	err := zip.openArchive(context.Background())
	require.NoError(t, err)

	// public/ public/index.html public/404.html public/symlink.html
	// public/subdir/ public/subdir/hello.html public/subdir/linked.html
	// public/bad_symlink.html public/subdir/2bp3Qzs...
	require.NotZero(t, zip.files)

	return zip, func() {
		zip.close()
		cleanup()
	}
}

func newZipFileServerURL(t *testing.T, zipFilePath string) (string, func()) {
	t.Helper()

	chdir := testhelpers.ChdirInPath(t, "../../../shared/pages", &chdirSet)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
	}))

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}
