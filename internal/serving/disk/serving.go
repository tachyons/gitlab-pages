package disk

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/zip"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var localVFS = vfs.Instrumented(local.VFS{}, "local")

var disk = &Disk{
	reader: Reader{
		fileSizeMetric: metrics.DiskServingFileSize,
		vfs: map[string]vfs.VFS{
			"":      localVFS,
			"local": localVFS,
			"zip":   vfs.Instrumented(zip.New(), "zip"),
		},
	},
}

// Disk describes a disk access serving
type Disk struct {
	reader Reader
}

// ServeFileHTTP serves a file from disk and returns true. It returns false
// when a file could not been found.
func (s *Disk) ServeFileHTTP(h serving.Handler) bool {
	return s.reader.tryFile(h) == nil
}

// ServeNotFoundHTTP tries to read a custom 404 page
func (s *Disk) ServeNotFoundHTTP(h serving.Handler) {
	if s.reader.tryNotFound(h) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(h.Writer)
}

// Instance returns a serving instance that is capable of reading files
// from the disk
func Instance() serving.Serving {
	return disk
}
