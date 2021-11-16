package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var errNotDirectory = errors.New("path needs to be a directory")

type VFS struct{}

func (localFs VFS) Root(ctx context.Context, path string, cacheKey string) (vfs.Root, error) {
	rootPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	rootPath, err = filepath.EvalSymlinks(rootPath)
	if err != nil {
		return nil, fmt.Errorf("could not evaluate symlinks: %w", err)
	}

	fi, err := os.Lstat(rootPath)
	if err != nil {
		return nil, err
	}

	if !fi.Mode().IsDir() {
		return nil, errNotDirectory
	}

	return &Root{rootPath: rootPath}, nil
}

func (localFs *VFS) Name() string {
	return "local"
}

func (localFs *VFS) Reconfigure(*config.Config) error {
	// noop
	return nil
}
