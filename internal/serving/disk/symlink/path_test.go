// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package symlink_test

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var fs = vfs.Instrumented(&local.VFS{}, "local")

func tmpDir(t *testing.T) (vfs.Root, string, func()) {
	tmpDir, err := ioutil.TempDir("", "symlink_tests")
	require.NoError(t, err)

	root, err := fs.Root(context.Background(), tmpDir)
	require.NoError(t, err)

	return root, tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

type EvalSymlinksTest struct {
	// If dest is empty, the path is created; otherwise the dest is symlinked to the path.
	path, dest string
}

var EvalSymlinksTestDirs = []EvalSymlinksTest{
	{"test", ""},
	{"test/dir", ""},
	{"test/dir/link3", "../../"},
	{"test/link1", "../test"},
	{"test/link2", "dir"},
	{"test/linkabs", "/"},
	{"test/link4", "../test2"},
	{"test2", "test/dir"},
	// Issue 23444.
	{"src", ""},
	{"src/pool", ""},
	{"src/pool/test", ""},
	{"src/versions", ""},
	{"src/versions/current", "../../version"},
	{"src/versions/v1", ""},
	{"src/versions/v1/modules", ""},
	{"src/versions/v1/modules/test", "../../../pool/test"},
	{"version", "src/versions/v1"},
}

var EvalSymlinksTests = []EvalSymlinksTest{
	{"test", "test"},
	{"test/dir", "test/dir"},
	{"test/dir/../..", "."},
	{"test/link1", "test"},
	{"test/link2", "test/dir"},
	{"test/link1/dir", "test/dir"},
	{"test/link2/..", "test"},
	{"test/dir/link3", "."},
	{"test/link2/link3/test", "test"},
	{"test/linkabs", "/"},
	{"test/link4/..", "test"},
	{"src/versions/current/modules/test", "src/pool/test"},
}

// simpleJoin builds a file name from the directory and path.
// It does not use Join because we don't want ".." to be evaluated.
func simpleJoin(dir, path string) string {
	return dir + string(filepath.Separator) + path
}

func testEvalSymlinks(t *testing.T, rootPath, path, want string) {
	root, err := fs.Root(context.Background(), rootPath)
	require.NoError(t, err)

	have, err := symlink.EvalSymlinks(context.Background(), root, path)
	require.NoError(t, err)

	assert.Equal(t, filepath.Clean(want), filepath.Clean(have))
}

func TestEvalSymlinks(t *testing.T) {
	_, tmpDir, cleanup := tmpDir(t)
	defer cleanup()

	// Create the symlink farm using relative paths.
	for _, d := range EvalSymlinksTestDirs {
		var err error
		path := simpleJoin(tmpDir, d.path)
		if d.dest == "" {
			err = os.Mkdir(path, 0755)
		} else {
			err = os.Symlink(d.dest, path)
		}
		require.NoError(t, err)
	}

	// Evaluate the symlink farm.
	for _, test := range EvalSymlinksTests {
		t.Run(test.path, func(t *testing.T) {
			testEvalSymlinks(t, tmpDir, test.path, test.dest)

			// test EvalSymlinks(".")
			testEvalSymlinks(t, simpleJoin(tmpDir, test.path), ".", ".")

			// test EvalSymlinks("C:.") on Windows
			if runtime.GOOS == "windows" {
				volDot := filepath.VolumeName(tmpDir) + "."
				testEvalSymlinks(t, simpleJoin(tmpDir, test.path), volDot, volDot)
			}

			// test EvalSymlinks(".."+path)
			dotdotPath := simpleJoin("..", test.dest)
			if filepath.IsAbs(test.dest) || os.IsPathSeparator(test.dest[0]) {
				dotdotPath = test.dest
			}
			testEvalSymlinks(t,
				simpleJoin(tmpDir, "test"),
				simpleJoin("..", test.path),
				dotdotPath)

			// test EvalSymlinks(p) where p is relative path
			testEvalSymlinks(t, tmpDir, test.path, test.dest)
		})
	}
}

func TestEvalSymlinksIsNotExist(t *testing.T) {
	root, tmpDir, cleanup := tmpDir(t)
	defer cleanup()

	_, err := symlink.EvalSymlinks(context.Background(), root, "notexist")
	require.True(t, os.IsNotExist(err), "file is not found")

	err = os.Symlink("notexist", filepath.Join(tmpDir, "link"))
	require.NoError(t, err)

	_, err = symlink.EvalSymlinks(context.Background(), root, "link")
	require.True(t, os.IsNotExist(err), "file is not found")
}

func TestIssue13582(t *testing.T) {
	root, tmpDir, cleanup := tmpDir(t)
	defer cleanup()

	dir := filepath.Join(tmpDir, "dir")
	err := os.Mkdir(dir, 0755)
	require.NoError(t, err)

	linkToDir := filepath.Join(tmpDir, "link_to_dir")
	err = os.Symlink(dir, linkToDir)
	require.NoError(t, err)

	file := filepath.Join(linkToDir, "file")
	err = ioutil.WriteFile(file, nil, 0644)
	require.NoError(t, err)

	link1 := filepath.Join(linkToDir, "link1")
	err = os.Symlink(file, link1)
	require.NoError(t, err)

	link2 := filepath.Join(linkToDir, "link2")
	err = os.Symlink(link1, link2)
	require.NoError(t, err)

	tests := []struct {
		path, want string
	}{
		{"dir", "dir"},
		{"link_to_dir", "dir"},
		{"link_to_dir/file", "dir/file"},
		{"link_to_dir/link1", "dir/file"},
		{"link_to_dir/link2", "dir/file"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			have, err := symlink.EvalSymlinks(context.Background(), root, test.path)
			require.NoError(t, err)
			assert.Equal(t, test.want, have)
		})
	}
}
