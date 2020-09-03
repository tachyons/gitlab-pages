package vfs

import (
	"context"
	"strconv"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VFS abstracts the things Pages needs to serve a static site from disk.
type VFS interface {
	Root(ctx context.Context, path string) (Root, error)
}

func Instrumented(fs VFS, name string) VFS {
	return &instrumentedVFS{fs: fs, name: name}
}

type instrumentedVFS struct {
	fs   VFS
	name string
}

func (i *instrumentedVFS) increment(operation string, err error) {
	metrics.VFSOperations.WithLabelValues(i.name, operation, strconv.FormatBool(err == nil)).Inc()
}

func (i *instrumentedVFS) log() *log.Entry {
	return log.WithField("vfs", i.name)
}

func (i *instrumentedVFS) Root(ctx context.Context, path string) (Root, error) {
	root, err := i.fs.Root(ctx, path)

	i.increment("Root", err)
	i.log().
		WithField("path", path).
		WithError(err).
		Traceln("Root call")

	if err != nil {
		return nil, err
	}

	return &instrumentedRoot{root: root, name: i.name, rootPath: path}, nil
}
