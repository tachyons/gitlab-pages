package symlink

import (
	"context"
	"path/filepath"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

func volumeNameLen(s string) int { return 0 }

func IsAbs(path string) bool   { return filepath.IsAbs(path) }
func Clean(path string) string { return filepath.Clean(path) }

func EvalSymlinks(ctx context.Context, root vfs.Root, path string) (string, error) {
	return walkSymlinks(ctx, root, path)
}
