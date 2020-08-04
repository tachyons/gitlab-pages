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

type VFS struct {
	Root string
}

func (fs VFS) path(name string) (string, error) {
	if strings.HasPrefix(filepath.Clean(name), "../") {
		return "", fmt.Errorf("vfs.File: invalid path %q", name)
	}
	return fs.Root + "/" + name, nil
}

func (fs VFS) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	path, err := fs.path(name)
	if err != nil {
		return nil, err
	}

	return os.Lstat(path)
}

func (fs VFS) Readlink(ctx context.Context, name string) (string, error) {
	path, err := fs.path(name)
	if err != nil {
		return "", err
	}

	return os.Readlink(path)
}

func (fs VFS) Open(ctx context.Context, name string) (vfs.File, error) {
	path, err := fs.path(name)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
