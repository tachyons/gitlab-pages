package zip

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httpfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/lru"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
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

type lruCache interface {
	FindOrFetch(cacheNamespace, key string, fetchFn func() (interface{}, error)) (interface{}, error)
}

// zipVFS is a simple cached implementation of the vfs.VFS interface
type zipVFS struct {
	cache     *cache.Cache
	cacheLock sync.Mutex

	openTimeout             time.Duration
	cacheExpirationInterval time.Duration
	cacheRefreshInterval    time.Duration
	cacheCleanupInterval    time.Duration

	dataOffsetCache lruCache
	readlinkCache   lruCache

	// the `int64` needs to be 64bit aligned on some 32bit systems
	// https://gitlab.com/gitlab-org/gitlab/-/issues/337261
	archiveCount *int64
	httpClient   *http.Client
}

// New creates a zipVFS instance that can be used by a serving request
func New(cfg *config.ZipServing) vfs.VFS {
	zipVFS := &zipVFS{
		cacheExpirationInterval: cfg.ExpirationInterval,
		cacheRefreshInterval:    cfg.RefreshInterval,
		cacheCleanupInterval:    cfg.CleanupInterval,
		openTimeout:             cfg.OpenTimeout,
		httpClient: &http.Client{
			// TODO: make this timeout configurable
			// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/457
			Timeout: 30 * time.Minute,
			Transport: httptransport.NewMeteredRoundTripper(
				httptransport.NewTransport(),
				"zip_vfs",
				metrics.HTTPRangeTraceDuration,
				metrics.HTTPRangeRequestDuration,
				metrics.HTTPRangeRequestsTotal,
				httptransport.DefaultTTFBTimeout,
			),
		},
		archiveCount: new(int64),
	}

	zipVFS.resetCache()

	// TODO: To be removed with https://gitlab.com/gitlab-org/gitlab-pages/-/issues/480
	zipVFS.dataOffsetCache = lru.New(
		"data-offset",
		lru.WithMaxSize(defaultDataOffsetItems),
		lru.WithExpirationInterval(defaultDataOffsetExpirationInterval),
		lru.WithCachedEntriesMetric(metrics.ZipCachedEntries),
		lru.WithCachedRequestsMetric(metrics.ZipCacheRequests),
	)
	zipVFS.readlinkCache = lru.New(
		"readlink",
		lru.WithMaxSize(defaultReadlinkItems),
		lru.WithExpirationInterval(defaultReadlinkExpirationInterval),
		lru.WithCachedEntriesMetric(metrics.ZipCachedEntries),
		lru.WithCachedRequestsMetric(metrics.ZipCacheRequests),
	)

	return zipVFS
}

// Reconfigure will update the zipVFS configuration values and will reset the
// cache
func (fs *zipVFS) Reconfigure(cfg *config.Config) error {
	fs.cacheLock.Lock()
	defer fs.cacheLock.Unlock()

	fs.openTimeout = cfg.Zip.OpenTimeout
	fs.cacheExpirationInterval = cfg.Zip.ExpirationInterval
	fs.cacheRefreshInterval = cfg.Zip.RefreshInterval
	fs.cacheCleanupInterval = cfg.Zip.CleanupInterval

	if err := fs.reconfigureTransport(cfg); err != nil {
		return err
	}

	fs.resetCache()

	return nil
}

func (fs *zipVFS) reconfigureTransport(cfg *config.Config) error {
	fsTransport, err := httpfs.NewFileSystemPath(cfg.Zip.AllowedPaths)
	if err != nil {
		return err
	}

	fs.httpClient.Transport.(httptransport.Transport).
		RegisterProtocol("file", http.NewFileTransport(fsTransport))

	return nil
}

func (fs *zipVFS) resetCache() {
	fs.cache = cache.New(fs.cacheExpirationInterval, fs.cacheCleanupInterval)
	fs.cache.OnEvicted(func(s string, i interface{}) {
		metrics.ZipCachedEntries.WithLabelValues("archive").Dec()

		i.(*zipArchive).onEvicted()
	})
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

		// If archive is not found, return a known `vfs` error
		if err == httprange.ErrNotFound {
			err = &vfs.ErrNotExist{Inner: err}
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
func (fs *zipVFS) findOrCreateArchive(ctx context.Context, key string) (*zipArchive, error) {
	// This needs to happen in lock to ensure that
	// concurrent access will not remove it
	// it is needed due to the bug https://github.com/patrickmn/go-cache/issues/48
	fs.cacheLock.Lock()
	defer fs.cacheLock.Unlock()

	archive, expiry, found := fs.cache.GetWithExpiration(key)
	if found {
		status, _ := archive.(*zipArchive).openStatus()
		switch status {
		case archiveOpening:
			metrics.ZipCacheRequests.WithLabelValues("archive", "hit-opening").Inc()

		case archiveOpenError:
			// this means that archive is likely corrupted
			// we keep it for duration of cache entry expiry (negative cache)
			metrics.ZipCacheRequests.WithLabelValues("archive", "hit-open-error").Inc()

		case archiveOpened:
			if time.Until(expiry) < fs.cacheRefreshInterval {
				fs.cache.SetDefault(key, archive)
				metrics.ZipCacheRequests.WithLabelValues("archive", "hit-refresh").Inc()
			} else {
				metrics.ZipCacheRequests.WithLabelValues("archive", "hit").Inc()
			}

		case archiveCorrupted:
			// this means that archive is likely changed
			// we should invalidate it immediately
			metrics.ZipCacheRequests.WithLabelValues("archive", "corrupted").Inc()
			archive = nil
		}
	}

	if archive == nil {
		archive = newArchive(fs, fs.openTimeout)

		// We call delete to ensure that expired item
		// is properly evicted as there's a bug in a cache library:
		// https://github.com/patrickmn/go-cache/issues/48
		fs.cache.Delete(key)

		// if adding the archive to the cache fails it means it's already been added before
		// this is done to find concurrent additions.
		if fs.cache.Add(key, archive, fs.cacheExpirationInterval) != nil {
			metrics.ZipCacheRequests.WithLabelValues("archive", "already-cached").Inc()
			return nil, errAlreadyCached
		}

		metrics.ZipCacheRequests.WithLabelValues("archive", "miss").Inc()
		metrics.ZipCachedEntries.WithLabelValues("archive").Inc()
	}

	return archive.(*zipArchive), nil
}

// findOrOpenArchive gets archive from cache and tries to open it
func (fs *zipVFS) findOrOpenArchive(ctx context.Context, key, path string) (*zipArchive, error) {
	zipArchive, err := fs.findOrCreateArchive(ctx, key)
	if err != nil {
		return nil, err
	}

	err = zipArchive.openArchive(ctx, path)
	if err != nil {
		return nil, err
	}

	return zipArchive, nil
}
