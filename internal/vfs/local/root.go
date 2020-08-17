package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

type Root struct {
	path string
}

func (r *Root) fullPath(path string) string {
	return filepath.Join(r.path, path)
}

func (r *Root) validateFullPath(fullPath string) (string, string, error) {
	if r.path == fullPath {
		return fullPath, "", nil
	}

	subPath := strings.TrimPrefix(fullPath, r.path+"/")

	// The requested path resolved to somewhere outside of the `r.path` directory
	if fullPath == subPath {
		return "", "", fmt.Errorf("%q should be in %q", fullPath, r.path)
	}

	return fullPath, subPath, nil
}

func (r *Root) sanitizePath(path string) (string, string, error) {
	return r.validateFullPath(r.fullPath(path))
}

func (r *Root) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	fullPath, _, err := r.sanitizePath(name)
	if err != nil {
		return nil, err
	}

	return os.Lstat(fullPath)
}

func (r *Root) Readlink(ctx context.Context, name string) (string, error) {
	fullPath, _, err := r.sanitizePath(name)
	if err != nil {
		return "", err
	}

	target, err := os.Readlink(fullPath)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(target) {
		return filepath.Rel(filepath.Dir(fullPath), target)
	}

	return target, nil
}

func (r *Root) Open(ctx context.Context, name string) (vfs.File, error) {
	fullPath, _, err := r.sanitizePath(name)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(fullPath, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
