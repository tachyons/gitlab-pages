package local

import (
	"context"
	"path/filepath"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

type VFS struct{}

func (fs VFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	realPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	realPath, err = filepath.EvalSymlinks(realPath)
	if err != nil {
		return nil, err
	}

	return &Root{path: realPath}, nil
}
