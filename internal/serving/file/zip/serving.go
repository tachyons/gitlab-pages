package zip

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/file"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/zip"
)

var zipServing = file.New(vfs.Instrumented(zip.New(), "zip"))

// New returns a serving instance that is capable of reading files
// from the disk
func New() serving.Serving {
	return zipServing
}
