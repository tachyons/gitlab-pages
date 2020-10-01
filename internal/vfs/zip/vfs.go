package zip

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	// TODO: make these configurable https://gitlab.com/gitlab-org/gitlab-pages/-/issues/464
	defaultCacheExpirationInterval = time.Minute
	defaultCacheCleanupInterval    = time.Minute / 2
	defaultCacheRefreshInterval    = time.Minute / 2
)

var (
	errAlreadyCached = errors.New("archive already cached")
)

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cache *cache.Cache
}

// New creates a zipVFS instance that can be used by a serving request
func New() vfs.VFS {
	zipVFS := &zipVFS{
		// TODO: add cache operation callbacks https://gitlab.com/gitlab-org/gitlab-pages/-/issues/465
		cache: cache.New(defaultCacheExpirationInterval, defaultCacheCleanupInterval),
	}

	zipVFS.cache.OnEvicted(func(s string, i interface{}) {
		metrics.ZipCachedArchives.Dec()

		i.(*zipArchive).onEvicted()
	})

	return zipVFS
}

// Root opens an archive given a URL path and returns an instance of zipArchive
// that implements the vfs.VFS interface.
// To avoid using locks, the findOrOpenArchive function runs inside of a for
// loop until an archive is either found or created and saved.
// If findOrOpenArchive returns errAlreadyCached, the for loop will continue
// to try and find the cached archive or return if there's an error, for example
// if the context is canceled.
func (fs *zipVFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	urlPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// we do it in loop to not use any additional locks
	for {
		root, err := fs.findOrOpenArchive(ctx, urlPath.String())
		if err == errAlreadyCached {
			continue
		}

		return root, err
	}
}

func (fs *zipVFS) Name() string {
	return "zip"
}

// findOrOpenArchive if found in fs.cache refresh if needed and return it.
// otherwise open the archive and try to save it, if saving fails it's because
// the archive has already been cached (e.g. by another concurrent request)
func (fs *zipVFS) findOrOpenArchive(ctx context.Context, path string) (*zipArchive, error) {
	archive, expiry, found := fs.cache.GetWithExpiration(path)
	if found {
		metrics.ZipServingArchiveCache.WithLabelValues("hit").Inc()

		// TODO: do not refreshed errored archives https://gitlab.com/gitlab-org/gitlab-pages/-/merge_requests/351
		if time.Until(expiry) < defaultCacheRefreshInterval {
			// refresh item
			fs.cache.SetDefault(path, archive)
		}
	} else {
		archive = newArchive(path, DefaultOpenTimeout)

		// if adding the archive to the cache fails it means it's already been added before
		// this is done to find concurrent additions.
		if fs.cache.Add(path, archive, cache.DefaultExpiration) != nil {
			return nil, errAlreadyCached
		}
		metrics.ZipServingArchiveCache.WithLabelValues("miss").Inc()
		metrics.ZipCachedArchives.Inc()
	}

	zipArchive := archive.(*zipArchive)

	err := zipArchive.openArchive(ctx)
	if err != nil {
		return nil, err
	}

	return zipArchive, nil
}
