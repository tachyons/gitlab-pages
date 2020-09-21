package zip

import (
	"context"
	"errors"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	// TODO: make these configurable https://gitlab.com/gitlab-org/gitlab-pages/-/issues/464
	cacheExpirationInterval = time.Minute
	cacheRefreshInterval    = time.Minute / 2
)

var (
	errNotZipArchive = errors.New("cached item is not a zip archive")
	errAlreadyCached = errors.New("archive already cached")
)

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cache *cache.Cache
}

// Root opens an archive given a URL path and returns an instance of zipArchive
// that implements the vfs.VFS interface.
// To avoid using locks, the findOrOpenArchive function runs inside of a for
// loop until an archive is either found or created and saved.
// If findOrOpenArchive returns errAlreadyCached, the for loop will continue
// to try and find the cached archive or return if there's an error, for example
// if the context is canceled.
func (fs *zipVFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	// we do it in loop to not use any additional locks
	for {
		root, err := fs.findOrOpenArchive(ctx, path)
		if err == errAlreadyCached {
			continue
		}

		return root, err
	}
}

// New creates a zipVFS instance that can be used by a serving request
func New() vfs.VFS {
	return &zipVFS{
		// TODO: add cache operation callbacks https://gitlab.com/gitlab-org/gitlab-pages/-/issues/465
		cache: cache.New(cacheExpirationInterval, cacheRefreshInterval),
	}
}

// findOrOpenArchive if found in fs.cache refresh if needed and return it.
// otherwise open the archive and try to save it, if saving fails it's because
// the archive has already been cached (e.g. by another request)
func (fs *zipVFS) findOrOpenArchive(ctx context.Context, path string) (*zipArchive, error) {
	archive, expiry, found := fs.cache.GetWithExpiration(path)
	if found {
		if time.Until(expiry) < cacheRefreshInterval {
			// refresh item
			fs.cache.Set(path, archive, cache.DefaultExpiration)
		}
	} else {
		archive = newArchive(path, DefaultOpenTimeout)

		// if adding the archive to the cache fails it means it's already been added before
		if fs.cache.Add(path, archive, cache.DefaultExpiration) != nil {
			return nil, errAlreadyCached
		}
	}

	zipDir, ok := archive.(*zipArchive)
	if !ok {
		// fail if the found archive in cache is not a zipArchive (just for type safety)
		return nil, errNotZipArchive
	}

	err := zipDir.openArchive(ctx)
	return zipDir, err
}
