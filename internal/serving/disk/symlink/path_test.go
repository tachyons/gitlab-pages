// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filepath_test

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	filepath "gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
)

func chtmpdir(t *testing.T) (restore func()) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("chtmpdir: %v", err)
	}
	d, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("chtmpdir: %v", err)
	}
	if err := os.Chdir(d); err != nil {
		t.Fatalf("chtmpdir: %v", err)
	}
	return func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("chtmpdir: %v", err)
		}
		os.RemoveAll(d)
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

func testEvalSymlinks(t *testing.T, path, want string) {
	have, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Errorf("EvalSymlinks(%q) error: %v", path, err)
		return
	}
	if filepath.Clean(have) != filepath.Clean(want) {
		t.Errorf("EvalSymlinks(%q) returns %q, want %q", path, have, want)
	}
}

func testEvalSymlinksAfterChdir(t *testing.T, wd, path, want string) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Chdir(cwd)
		if err != nil {
			t.Fatal(err)
		}
	}()

	err = os.Chdir(wd)
	if err != nil {
		t.Fatal(err)
	}

	have, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Errorf("EvalSymlinks(%q) in %q directory error: %v", path, wd, err)
		return
	}
	if filepath.Clean(have) != filepath.Clean(want) {
		t.Errorf("EvalSymlinks(%q) in %q directory returns %q, want %q", path, wd, have, want)
	}
}

func TestEvalSymlinks(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "evalsymlink")
	if err != nil {
		t.Fatal("creating temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	// /tmp may itself be a symlink! Avoid the confusion, although
	// it means trusting the thing we're testing.
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal("eval symlink for tmp dir:", err)
	}

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
		path := simpleJoin(tmpDir, test.path)

		dest := simpleJoin(tmpDir, test.dest)
		if filepath.IsAbs(test.dest) || os.IsPathSeparator(test.dest[0]) {
			dest = test.dest
		}
		testEvalSymlinks(t, path, dest)

		// test EvalSymlinks(".")
		testEvalSymlinksAfterChdir(t, path, ".", ".")

		// test EvalSymlinks("C:.") on Windows
		if runtime.GOOS == "windows" {
			volDot := filepath.VolumeName(tmpDir) + "."
			testEvalSymlinksAfterChdir(t, path, volDot, volDot)
		}

		// test EvalSymlinks(".."+path)
		dotdotPath := simpleJoin("..", test.dest)
		if filepath.IsAbs(test.dest) || os.IsPathSeparator(test.dest[0]) {
			dotdotPath = test.dest
		}
		testEvalSymlinksAfterChdir(t,
			simpleJoin(tmpDir, "test"),
			simpleJoin("..", test.path),
			dotdotPath)

		// test EvalSymlinks(p) where p is relative path
		testEvalSymlinksAfterChdir(t, tmpDir, test.path, test.dest)
	}
}

func TestEvalSymlinksIsNotExist(t *testing.T) {
	defer chtmpdir(t)()

	_, err := filepath.EvalSymlinks("notexist")
	if !os.IsNotExist(err) {
		t.Errorf("expected the file is not found, got %v\n", err)
	}

	err = os.Symlink("notexist", "link")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("link")

	_, err = filepath.EvalSymlinks("link")
	if !os.IsNotExist(err) {
		t.Errorf("expected the file is not found, got %v\n", err)
	}
}

func TestIssue13582(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "issue13582")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dir := filepath.Join(tmpDir, "dir")
	err = os.Mkdir(dir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	linkToDir := filepath.Join(tmpDir, "link_to_dir")
	err = os.Symlink(dir, linkToDir)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(linkToDir, "file")
	err = ioutil.WriteFile(file, nil, 0644)
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

	// /tmp may itself be a symlink!
	realTmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	realDir := filepath.Join(realTmpDir, "dir")
	realFile := filepath.Join(realDir, "file")

	tests := []struct {
		path, want string
	}{
		{dir, realDir},
		{linkToDir, realDir},
		{file, realFile},
		{link1, realFile},
		{link2, realFile},
	}
	for i, test := range tests {
		have, err := filepath.EvalSymlinks(test.path)
		if err != nil {
			t.Fatal(err)
		}
		if have != test.want {
			t.Errorf("test#%d: EvalSymlinks(%q) returns %q, want %q", i, test.path, have, test.want)
		}
	}
}
