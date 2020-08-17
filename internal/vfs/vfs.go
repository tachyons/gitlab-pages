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
	Dir(ctx context.Context, path string) (Dir, error)
}

// Dir abstracts the things Pages needs to serve a static site from a given path.
type Dir interface {
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

func (i *InstrumentedVFS) Dir(ctx context.Context, path string) (Dir, error) {
	dir, err := i.fs.Dir(ctx, path)
	i.increment("Lstat", err)
	if dir != nil {
		dir = &InstrumentedDir{dir: dir, name: i.name}
	}
	return dir, err
}

type InstrumentedDir struct {
	dir  Dir
	name string
}

func (i *InstrumentedDir) increment(operation string, err error) {
	metrics.VFSOperations.WithLabelValues(i.name, operation, strconv.FormatBool(err == nil)).Inc()
}

func (i *InstrumentedDir) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	fi, err := i.dir.Lstat(ctx, name)
	i.increment("Lstat", err)
	return fi, err
}

func (i *InstrumentedDir) Readlink(ctx context.Context, name string) (string, error) {
	target, err := i.dir.Readlink(ctx, name)
	i.increment("Readlink", err)
	return target, err
}

func (i *InstrumentedDir) Open(ctx context.Context, name string) (File, error) {
	f, err := i.dir.Open(ctx, name)
	i.increment("Open", err)
	return f, err
}
