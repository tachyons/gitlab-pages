package vfs

import (
	"context"
	"io"
	"os"
	"strconv"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VFS abstracts the things Pages needs to serve a static site from disk.
type VFS interface {
	Lstat(ctx context.Context, name string) (os.FileInfo, error)
	Readlink(ctx context.Context, name string) (string, error)
	Open(ctx context.Context, name string) (File, error)
}

// File represents an open file, which will typically be the response body of a Pages request.
type File interface {
	io.Reader
	io.Seeker
	io.Closer
}

func Instrumented(fs VFS, name string) VFS {
	return &InstrumentedVFS{fs: fs, name: name}
}

type InstrumentedVFS struct {
	fs   VFS
	name string
}

func (i *InstrumentedVFS) increment(operation string, err error) {
	metrics.VFSOperations.WithLabelValues(i.name, operation, strconv.FormatBool(err == nil)).Inc()
}

func (i *InstrumentedVFS) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	fi, err := i.fs.Lstat(ctx, name)
	i.increment("Lstat", err)
	return fi, err
}

func (i *InstrumentedVFS) Readlink(ctx context.Context, name string) (string, error) {
	target, err := i.fs.Readlink(ctx, name)
	i.increment("Readlink", err)
	return target, err
}

func (i *InstrumentedVFS) Open(ctx context.Context, name string) (File, error) {
	f, err := i.fs.Open(ctx, name)
	i.increment("Open", err)
	return f, err
}
