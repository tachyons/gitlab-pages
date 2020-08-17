package zip

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const cacheExpirationInterval = time.Minute
const cacheRefreshInterval = time.Minute / 2
const cacheEvictInterval = time.Minute

type zipVFS struct {
	cache *cache.Cache
}

func (fs *zipVFS) Dir(ctx context.Context, path string) (vfs.Dir, error) {
	// we do it in loop to not use any additional locks
	for {
		dir, till, found := fs.cache.GetWithExpiration(path)
		if found {
			if till.Sub(time.Now()) < cacheRefreshInterval {
				// refresh item
				fs.cache.Set(path, dir, cache.DefaultExpiration)
			}
		} else {
			dir = newArchive(path)

			// if it errors, it means that it is already added
			// retry again to get it
			if fs.cache.Add(path, dir, cache.DefaultExpiration) != nil {
				continue
			}
		}

		zipDir := dir.(*zipArchive)

		err := zipDir.openArchive(ctx)
		return zipDir, err
	}
}

func New() vfs.VFS {
	vfs := &zipVFS{
		cache: cache.New(cacheExpirationInterval, cacheRefreshInterval),
	}

	vfs.cache.OnEvicted(func(path string, object interface{}) {
		if archive, ok := object.(*zipArchive); archive != nil && ok {
			archive.close()
		}
	})
	return vfs
}
