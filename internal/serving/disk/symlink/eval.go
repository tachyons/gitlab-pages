package symlink

import (
	"context"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

// EvalSymlinks does the same thing as stdlib filepath.EvalSymlinks, except it goes through a vfs.VFS interface.
func EvalSymlinks(ctx context.Context, fs vfs.VFS, path string) (string, error) {
	return walkSymlinks(ctx, fs, path)
}

// volumeNameLen has a non-trivial implementation only on Windows
func volumeNameLen(path string) int { return 0 }
