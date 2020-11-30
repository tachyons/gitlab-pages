package zip

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/zip"
)

var instance = disk.New(vfs.Instrumented(zip.New(&config.ZipServing{})))

// Instance returns a serving instance that is capable of reading files
// from a zip archives opened from a URL, most likely stored in object storage
func Instance() serving.Serving {
	return instance
}
