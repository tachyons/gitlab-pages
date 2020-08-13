package local

import (
	"context"
	"os"

	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

type VFS struct{}

func (fs VFS) Lstat(ctx context.Context, name string) (os.FileInfo, error) { return os.Lstat(name) }
func (fs VFS) Readlink(ctx context.Context, name string) (string, error)   { return os.Readlink(name) }

func (fs VFS) Open(ctx context.Context, name string) (vfs.File, error) {
	return os.OpenFile(name, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
