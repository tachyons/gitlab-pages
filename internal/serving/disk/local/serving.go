package local

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var instance = disk.New(vfs.Instrumented(&local.VFS{}))

// Instance returns a serving instance that is capable of reading files
// from the disk
func Instance() serving.Serving {
	return instance
}
