// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package symlink_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var localFs = vfs.Instrumented(&local.VFS{})

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
func simpleJoin(path ...string) string {
	return strings.Join(path, string(filepath.Separator))
}

func testEvalSymlinks(t *testing.T, wd, path, want string) {
	root, err := localFs.Root(context.Background(), wd, "")
	require.NoError(t, err)

	have, err := symlink.EvalSymlinks(context.Background(), root, path)
	if err != nil {
		t.Errorf("evalSymlinks(%q) error: %v", path, err)
		return
	}
	if filepath.Clean(have) != filepath.Clean(want) {
		t.Errorf("evalSymlinks(%q) returns %q, want %q", path, have, want)
	}
}

func TestEvalSymlinks(t *testing.T) {
	_, tmpDir := testhelpers.TmpDir(t)

	// Create the symlink farm using relative paths.
	for _, d := range EvalSymlinksTestDirs {
		var err error
		path := simpleJoin(tmpDir, d.path)
		if d.dest == "" {
			err = os.Mkdir(path, 0755)
		} else {
			err = os.Symlink(d.dest, path)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Evaluate the symlink farm.
	for _, test := range EvalSymlinksTests {
		testEvalSymlinks(t, tmpDir, test.path, test.dest)

		// test EvalSymlinks(".")
		testEvalSymlinks(t, simpleJoin(tmpDir, test.path), ".", ".")

		// test EvalSymlinks("C:.") on Windows
		if runtime.GOOS == "windows" {
			volDot := filepath.VolumeName(tmpDir) + "."
			testEvalSymlinks(t, simpleJoin(tmpDir, test.path), volDot, volDot)
		}

		// test EvalSymlinks(".."+path)
		testEvalSymlinks(t,
			tmpDir,
			simpleJoin("test", "..", test.path),
			test.dest)
	}
}

func TestEvalSymlinksIsNotExist(t *testing.T) {
	root, _ := testhelpers.TmpDir(t)

	_, err := symlink.EvalSymlinks(context.Background(), root, "notexist")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected the file is not found, got %v\n", err)
	}

	err = os.Symlink("notexist", "link")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("link")

	_, err = symlink.EvalSymlinks(context.Background(), root, "link")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected the file is not found, got %v\n", err)
	}
}

func TestIssue13582(t *testing.T) {
	root, tmpDir := testhelpers.TmpDir(t)

	dir := filepath.Join(tmpDir, "dir")
	err := os.Mkdir(dir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	linkToDir := filepath.Join(tmpDir, "link_to_dir")
	err = os.Symlink(dir, linkToDir)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(linkToDir, "file")
	err = os.WriteFile(file, nil, 0644)
	if err != nil {
		t.Fatal(err)
	}
	link1 := filepath.Join(linkToDir, "link1")
	err = os.Symlink(file, link1)
	if err != nil {
		t.Fatal(err)
	}
	link2 := filepath.Join(linkToDir, "link2")
	err = os.Symlink(link1, link2)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path, want string
	}{
		{"dir", "dir"},
		{"link_to_dir", "dir"},
		{"link_to_dir/file", "dir/file"},
		{"link_to_dir/link1", "dir/file"},
		{"link_to_dir/link2", "dir/file"},
	}
	for i, test := range tests {
		have, err := symlink.EvalSymlinks(context.Background(), root, test.path)
		if err != nil {
			t.Fatal(err)
		}
		if have != test.want {
			t.Errorf("test#%d: EvalSymlinks(%q) returns %q, want %q", i, test.path, have, test.want)
		}
	}
}
