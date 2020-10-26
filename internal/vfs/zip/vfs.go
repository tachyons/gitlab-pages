package zip

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	// TODO: make these configurable https://gitlab.com/gitlab-org/gitlab-pages/-/issues/464
	defaultCacheExpirationInterval         = time.Minute
	defaultCacheCleanupInterval            = time.Minute / 2
	defaultCacheRefreshInterval            = time.Minute / 2
	defaultCacheNegativeExpirationInterval = 10 * time.Second

	// we assume that each item costs around 100 bytes
	// this gives around 5MB of raw memory needed without acceleration structures
	defaultDataOffsetItems              = 50000
	defaultDataOffsetExpirationInterval = time.Hour

	// we assume that each item costs around 200 bytes
	// this gives around 2MB of raw memory needed without acceleration structures
	defaultReadlinkItems              = 10000
	defaultReadlinkExpirationInterval = time.Hour
)

var (
	errAlreadyCached = errors.New("archive already cached")
)

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cache                   *cache.Cache
	cacheLock               sync.Mutex
	cacheExpirationInterval time.Duration
	cacheRefreshInterval    time.Duration
	cacheCleanupInterval    time.Duration

	dataOffsetCache *lruCache
	readlinkCache   *lruCache

	archiveCount int64
}

// Option function allows to override default values
type Option func(*zipVFS)

// WithCacheRefreshInterval when used it can override defaultCacheRefreshInterval
func WithCacheRefreshInterval(interval time.Duration) Option {
	return func(vfs *zipVFS) {
		vfs.cacheRefreshInterval = interval
	}
}

// WithCacheExpirationInterval when used it can override defaultCacheExpirationInterval
func WithCacheExpirationInterval(interval time.Duration) Option {
	return func(vfs *zipVFS) {
		vfs.cacheExpirationInterval = interval
	}
}

// WithCacheCleanupInterval when used it can override defaultCacheCleanupInterval
func WithCacheCleanupInterval(interval time.Duration) Option {
	return func(vfs *zipVFS) {
		vfs.cacheCleanupInterval = interval
	}
}

// New creates a zipVFS instance that can be used by a serving request
func New(options ...Option) vfs.VFS {
	zipVFS := &zipVFS{
		cacheExpirationInterval: defaultCacheExpirationInterval,
		cacheRefreshInterval:    defaultCacheRefreshInterval,
		cacheCleanupInterval:    defaultCacheCleanupInterval,
	}

	for _, option := range options {
		option(zipVFS)
	}

	zipVFS.cache = cache.New(zipVFS.cacheExpirationInterval, zipVFS.cacheCleanupInterval)
	zipVFS.cache.OnEvicted(func(s string, i interface{}) {
		metrics.ZipCachedEntries.WithLabelValues("archive").Dec()

		i.(*zipArchive).onEvicted()
	})

	// TODO: To be removed by https://gitlab.com/gitlab-org/gitlab-pages/-/merge_requests/375
	zipVFS.dataOffsetCache = newLruCache("data-offset", defaultDataOffsetItems, defaultDataOffsetExpirationInterval)
	zipVFS.readlinkCache = newLruCache("readlink", defaultReadlinkItems, defaultReadlinkExpirationInterval)

	return zipVFS
}

func (fs *zipVFS) keyFromPath(path string) (string, error) {
	// We assume that our URL is https://.../artifacts.zip?content-sign=aaa
	// our caching key is `https://.../artifacts.zip`
	// TODO: replace caching key with file_sha256
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/489
	key, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	key.RawQuery = ""
	key.Fragment = ""
	return key.String(), nil
}

// Root opens an archive given a URL path and returns an instance of zipArchive
// that implements the vfs.VFS interface.
// To avoid using locks, the findOrOpenArchive function runs inside of a for
// loop until an archive is either found or created and saved.
// If findOrOpenArchive returns errAlreadyCached, the for loop will continue
// to try and find the cached archive or return if there's an error, for example
// if the context is canceled.
func (fs *zipVFS) Root(ctx context.Context, path string) (vfs.Root, error) {
	key, err := fs.keyFromPath(path)
	if err != nil {
		return nil, err
	}

	// we do it in loop to not use any additional locks
	for {
		root, err := fs.findOrOpenArchive(ctx, key, path)
		if err == errAlreadyCached {
			continue
		}

		return root, err
	}
}

func (fs *zipVFS) Name() string {
	return "zip"
}

// findOrCreateArchive if found in fs.cache refresh if needed and return it.
// otherwise creates the archive entry in a cache and try to save it,
// if saving fails it's because the archive has already been cached
// (e.g. by another concurrent request)
func (fs *zipVFS) findOrCreateArchive(key string) (*zipArchive, error) {
	// This needs to happen in lock to ensure that
	// concurrent access will not remove it
	// it is needed due to the bug https://github.com/patrickmn/go-cache/issues/48
	fs.cacheLock.Lock()
	defer fs.cacheLock.Unlock()

	archive, expiry, found := fs.cache.GetWithExpiration(key)
	if found && archive.(*zipArchive).isValid() {
		if time.Until(expiry) < fs.cacheRefreshInterval {
			// refresh valid item
			fs.cache.SetDefault(key, archive)
			metrics.ZipCacheRequests.WithLabelValues("archive", "hit-refresh").Inc()
		} else {
			metrics.ZipCacheRequests.WithLabelValues("archive", "hit").Inc()
		}
	} else {
		archive = newArchive(fs, DefaultOpenTimeout)

		// We call delete to ensure that expired item
		// is properly evicted as there's a bug in a cache library:
		// https://github.com/patrickmn/go-cache/issues/48
		fs.cache.Delete(key)

		// change expiration time if archive is invalid
		expiry := cache.DefaultExpiration
		if !archive.(*zipArchive).isValid() {
			expiry = defaultCacheNegativeExpirationInterval
		}

		// if adding the archive to the cache fails it means it's already been added before
		// this is done to find concurrent additions.
		if fs.cache.Add(key, archive, expiry) != nil {
			return nil, errAlreadyCached
		}

		if found {
			metrics.ZipCacheRequests.WithLabelValues("archive", "miss-invalid").Inc()
		} else {
			metrics.ZipCacheRequests.WithLabelValues("archive", "miss").Inc()
		}
		metrics.ZipCachedEntries.WithLabelValues("archive").Inc()
	}

	return archive.(*zipArchive), nil
}

// findOrOpenArchive gets archive from cache and tries to open it
func (fs *zipVFS) findOrOpenArchive(ctx context.Context, key, url string) (*zipArchive, error) {
	zipArchive, err := fs.findOrCreateArchive(key)
	if err != nil {
		return nil, err
	}

	err = zipArchive.openArchive(ctx, url)
	if err != nil {
		return nil, err
	}

	return zipArchive, nil
}
