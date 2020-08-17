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

type VFS struct{}

func (fs VFS) Dir(ctx context.Context, path string) (vfs.Dir, error) {
	return &Dir{path: filepath.Clean(path)}, nil
}

type Dir struct {
	path string
}

func (dir *Dir) validatePath(fullPath string) error {
	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, dir.path) && fullPath != dir.path {
		return fmt.Errorf("%q should be in %q", fullPath, dir.path)
	}

	return nil
}

func (dir *Dir) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	if err := dir.validatePath(name); err != nil {
		return nil, err
	}

	return os.Lstat(filepath.Join(dir.path, name))
}

func (dir *Dir) Readlink(ctx context.Context, name string) (string, error) {
	if err := dir.validatePath(name); err != nil {
		return "", err
	}

	return os.Readlink(filepath.Join(dir.path, name))
}

func (dir *Dir) Open(ctx context.Context, name string) (vfs.File, error) {
	if err := dir.validatePath(name); err != nil {
		return nil, err
	}

	return os.OpenFile(filepath.Join(dir.path, name), os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
