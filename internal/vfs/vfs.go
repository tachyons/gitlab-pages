package vfs

import (
	"context"
	"io"
	"os"
)

type VFS interface {
	Lstat(ctx context.Context, name string) (os.FileInfo, error)
	Readlink(ctx context.Context, name string) (string, error)
	Open(ctx context.Context, name string) (File, error)
}

type File interface {
	io.Reader
	io.Seeker
	io.Closer
}
