package zip

import (
	"context"
	"errors"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	// TODO: should we make zip root cache configurable?
	cacheExpirationInterval = time.Minute
	cacheRefreshInterval    = time.Minute / 2
)

var (
	errNotZipArchive = errors.New("cached item is not a zip archive")
)

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cache *cache.Cache
}

// Root opens an archive given a URL path and returns an instance of zipArchive
// that implements the vfs.Root.
// To avoid using locks, the function runs in a for loop until an archive is
// either found or created and saved. If there's an error while saving the
// archive to the given path because it already exists, the for loop will
// continue to try and find the already saved archive again.
func (fs *zipVFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	// we do it in loop to not use any additional locks
	for {
		archive, expiry, found := fs.cache.GetWithExpiration(path)
		if found {
			if time.Until(expiry) < cacheRefreshInterval {
				// refresh item
				fs.cache.Set(path, archive, cache.DefaultExpiration)
			}
		} else {
			archive = newArchive(path, DefaultOpenTimeout)

			// if it errors, it means that it is already added
			// retry again to get it
			// if adding the archive to the cache fails it means it's already been added before, retry getting archive
			if fs.cache.Add(path, archive, cache.DefaultExpiration) != nil {
				continue
			}
		}

		zipDir, ok := archive.(*zipArchive)
		if !ok {
			// fail if the found archive in cache is not a zipArchive
			return nil, errNotZipArchive
		}

		err := zipDir.openArchive(ctx)
		return zipDir, err
	}
}

// New creates a zipVFS instance that can be used by a serving request
func New() vfs.VFS {
	zipVFS := &zipVFS{
		cache: cache.New(cacheExpirationInterval, cacheRefreshInterval),
	}

	zipVFS.cache.OnEvicted(func(path string, object interface{}) {
		// TODO: Add and update zip metric on eviction https://gitlab.com/gitlab-org/gitlab-pages/-/issues/423
		if archive, ok := object.(*zipArchive); archive != nil && ok {
			archive.close()
		}
	})

	return zipVFS
}
