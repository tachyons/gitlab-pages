package zip

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

type zipVFS struct {
	cache *cache.Cache
}

func (fs *zipVFS) Dir(ctx context.Context, path string) (vfs.Dir, error) {
	// we do it in loop to not use any additional locks
	for {
		dir, found := fs.cache.Get(path)
		if !found {
			dir = newArchive(path)

			// if it errors, it means that it is already added
			// retry again to get it
			if fs.cache.Add(path, dir, cache.DefaultExpiration) != nil {
				continue
			}
		}

		err := dir.(*zipArchive).Open(ctx)
		return dir, err
	}
}

func New() vfs.VFS {
	vfs := &zipVFS{
		cache: cache.New(time.Minute, 2*time.Minute),
	}
	vfs.cache.OnEvicted(func(path string, object interface{}) {
		if archive, ok := object.(*Archive); archive != nil && ok {
			archive.Close()
		}
	})
	return vfs
}
