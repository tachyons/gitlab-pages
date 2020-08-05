package fileresolver

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenFileFromZip(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	tests := []struct {
		name            string
		archivePath     string
		subPath         string
		expectedContent string
		expectedErrMsg  string
	}{
		{
			name:            "file_exists_with_subpath_and_extension",
			archivePath:     "group/group.test.io/public.zip",
			subPath:         "index.html",
			expectedContent: "main-dir\n",
		},
		{
			name:            "file_exists_without_extension",
			archivePath:     "group/group.test.io/public.zip",
			subPath:         "index",
			expectedContent: "main-dir\n",
		},
		{
			name:            "file_exists_without_subpath",
			archivePath:     "group/group.test.io/public.zip",
			subPath:         "",
			expectedContent: "main-dir\n",
		},
		{
			name:           "file_does_not_exist_without_subpath",
			archivePath:    "group.no.projects/public.zip",
			subPath:        "",
			expectedErrMsg: "not found",
		},
		{
			name:           "file_does_not_exist",
			archivePath:    "group/group.test.io/public.zip",
			subPath:        "unknown_file.html",
			expectedErrMsg: "not found",
		},
		{
			name:            "symlink_inside_public",
			archivePath:     "group/symlink/public.zip",
			subPath:         "index.html",
			expectedContent: "group/symlink/public/content/index.html\n",
		},
	}

	z := z{
		maxSymlinkSize:  4096,
		maxSymlinkDepth: 3,
		zipDeployPath:   "public",
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := zip.OpenReader(tt.archivePath)
			require.NoError(t, err)
			defer reader.Close()
			z.archive = reader

			file, err := OpenFile("", tt.subPath, nil, z.resolvePublic)
			if tt.expectedErrMsg != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}
			require.NoError(t, err)

			content, err := ioutil.ReadAll(file)
			require.NoError(t, err)
			require.Contains(t, string(content), tt.expectedContent)
		})
	}
}

// const zipDeployPath = "public"
// const maxSymlinkSize = 4096
// const maxSymlinkDepth = 3

type z struct {
	archive *zip.ReadCloser

	zipDeployPath   string
	maxSymlinkSize  int64
	maxSymlinkDepth int
}

func (z *z) readSymlink(file *zip.File) (string, error) {
	fi := file.FileInfo()

	if (fi.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return "", nil
	}

	if fi.Size() > z.maxSymlinkSize {
		return "", errors.New("symlink size too long")
	}

	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// resolve symlink location relative to current file
	targetPath, err := filepath.Rel(filepath.Dir(file.Name), string(data))
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func (z *z) resolveUnchecked(path string) (*zip.File, error) {
	// limit the resolve depth of symlink
	for depth := 0; depth < z.maxSymlinkDepth; depth++ {
		file := z.find(path)
		if file == nil {
			break
		}

		targetPath, err := z.readSymlink(file)
		if err != nil {
			return nil, err
		}

		// not a symlink
		if targetPath == "" {
			return file, nil
		}

		path = targetPath
	}

	return nil, fmt.Errorf("%q: not found", path)
}

func (z *z) resolvePublic(path string) (io.ReadCloser, error) {
	path = filepath.Join(z.zipDeployPath, path)
	file, err := z.resolveUnchecked(path)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(file.Name, z.zipDeployPath+"/") {
		return nil, fmt.Errorf("%q: is not in %s/", file.Name, z.zipDeployPath)
	}

	return file.Open()
}

func (z *z) find(path string) *zip.File {
	if z.archive == nil {
		return nil
	}

	// This is O(n) search, very, very, very slow
	for _, file := range z.archive.File {
		if file.Name == path || file.Name == path+"/" {
			return file
		}
	}

	return nil
}
