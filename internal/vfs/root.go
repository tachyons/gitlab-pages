package vfs

import (
	"context"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Root abstracts the things Pages needs to serve a static site from a given root rootPath.
type Root interface {
	Lstat(ctx context.Context, name string) (os.FileInfo, error)
	Readlink(ctx context.Context, name string) (string, error)
	Open(ctx context.Context, name string) (File, error)
}

type instrumentedRoot struct {
	root     Root
	name     string
	rootPath string
}

func (i *instrumentedRoot) increment(operation string, err error) {
	metrics.VFSOperations.WithLabelValues(i.name, operation, strconv.FormatBool(err == nil)).Inc()
}

func (i *instrumentedRoot) log(ctx context.Context) *logrus.Entry {
	return log.ContextLogger(ctx).WithField("vfs", i.name).WithField("root-path", i.rootPath)
}

func (i *instrumentedRoot) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	fi, err := i.root.Lstat(ctx, name)

	i.increment("Lstat", err)
	i.log(ctx).
		WithField("name", name).
		WithError(err).
		Traceln("Lstat call")

	return fi, err
}

func (i *instrumentedRoot) Readlink(ctx context.Context, name string) (string, error) {
	target, err := i.root.Readlink(ctx, name)

	i.increment("Readlink", err)
	i.log(ctx).
		WithField("name", name).
		WithField("ret-target", target).
		WithError(err).
		Traceln("Readlink call")

	return target, err
}

func (i *instrumentedRoot) Open(ctx context.Context, name string) (File, error) {
	f, err := i.root.Open(ctx, name)

	i.increment("Open", err)
	i.log(ctx).
		WithField("name", name).
		WithError(err).
		Traceln("Open call")

	return f, err
}
