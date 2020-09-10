package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var errNotFile = errors.New("path needs to be a file")

type invalidPathError struct {
	rootPath      string
	requestedPath string
}

func (i *invalidPathError) Error() string {
	return fmt.Sprintf("%q should be in %q", i.requestedPath, i.rootPath)
}

type Root struct {
	rootPath string
}

func (r *Root) validatePath(path string) (string, string, error) {
	fullPath := filepath.Join(r.rootPath, path)

	if r.rootPath == fullPath {
		return fullPath, "", nil
	}

	vfsPath := strings.TrimPrefix(fullPath, r.rootPath+"/")

	// The requested path resolved to somewhere outside of the `r.rootPath` directory
	if fullPath == vfsPath {
		return "", "", &invalidPathError{rootPath: r.rootPath, requestedPath: fullPath}
	}

	return fullPath, vfsPath, nil
}

func (r *Root) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	fullPath, _, err := r.validatePath(name)
	if err != nil {
		return nil, err
	}

	return os.Lstat(fullPath)
}

func (r *Root) Readlink(ctx context.Context, name string) (string, error) {
	fullPath, _, err := r.validatePath(name)
	if err != nil {
		return "", err
	}

	target, err := os.Readlink(fullPath)
	if err != nil {
		return "", err
	}

	// If `target` is local to `rootPath` return relative
	if strings.HasPrefix(target, r.rootPath+"/") {
		return filepath.Rel(filepath.Dir(fullPath), target)
	}

	// If `target` is absolute return as-is making `EvalSymlinks`
	// to discover misuse of a root path
	if filepath.IsAbs(target) {
		return target, nil
	}

	// If `target` is relative, return as-is
	return target, nil
}

func (r *Root) Open(ctx context.Context, name string) (vfs.File, error) {
	fullPath, _, err := r.validatePath(name)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(fullPath, os.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	// We do a `Stat()` on a file due to race-conditions
	// Someone could update (unlikely) a file between `Stat()/Open()`
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fi.Mode().IsRegular() {
		file.Close()
		return nil, errNotFile
	}

	return file, nil
}
