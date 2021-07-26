package vfs

import (
	"context"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VFS abstracts the things Pages needs to serve a static site from disk.
type VFS interface {
	Root(ctx context.Context, path string, cacheKey string) (Root, error)
	Name() string
	Reconfigure(config *config.Config) error
}

func Instrumented(fs VFS) VFS {
	return &instrumentedVFS{fs: fs}
}

type instrumentedVFS struct {
	fs VFS
}

func (i *instrumentedVFS) increment(operation string, err error) {
	metrics.VFSOperations.WithLabelValues(i.fs.Name(), operation, strconv.FormatBool(err == nil)).Inc()
}

func (i *instrumentedVFS) log(ctx context.Context) *logrus.Entry {
	return log.ContextLogger(ctx).WithField("vfs", i.fs.Name())
}

func (i *instrumentedVFS) Root(ctx context.Context, path string, cacheKey string) (Root, error) {
	root, err := i.fs.Root(ctx, path, cacheKey)

	i.increment("Root", err)
	i.log(ctx).
		WithField("path", path).
		WithError(err).
		Traceln("Root call")

	if err != nil {
		return nil, err
	}

	return &instrumentedRoot{root: root, name: i.fs.Name(), rootPath: path}, nil
}

func (i *instrumentedVFS) Name() string {
	return i.fs.Name()
}

func (i *instrumentedVFS) Reconfigure(cfg *config.Config) error {
	return i.fs.Reconfigure(cfg)
}
