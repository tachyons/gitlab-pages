package gocloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob" // Azure support
	_ "gocloud.dev/blob/fileblob"  // local filesystem support
	_ "gocloud.dev/blob/gcsblob"   // GCS support
	_ "gocloud.dev/blob/s3blob"    // S3 support

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	fileModeKey      = "gitlab-pages-filemode"
	symlinkTargetKey = "gitlab-pages-symlink"
)

type VFS struct {
	bucket *blob.Bucket
	cache  *lru.Cache
	sync.Mutex
}

var _ = vfs.VFS(&VFS{}) // compile time assertion that *VFS satisfies vfs.VFS

func New(ctx context.Context, url string, prefix string) (*VFS, error) {
	b, err := blob.OpenBucket(ctx, url)
	if err != nil {
		return nil, err
	}

	if prefix != "" {
		b = blob.PrefixedBucket(b, prefix)
	}

	const cacheSize = 100000
	return &VFS{
		bucket: b,
		cache:  lru.New(cacheSize),
	}, err
}

type Attributes struct {
	key           string
	size          int64
	mode          os.FileMode
	modTime       time.Time
	symlinkTarget string
}

func (a *Attributes) Name() string       { return a.key }
func (a *Attributes) Size() int64        { return a.size }
func (a *Attributes) Mode() os.FileMode  { return a.mode }
func (a *Attributes) ModTime() time.Time { return a.modTime }
func (a *Attributes) IsDir() bool        { return a.mode.IsDir() }
func (a *Attributes) IsSymlink() bool    { return (a.mode & os.ModeType) == os.ModeSymlink }
func (a *Attributes) Sys() interface{}   { return nil }

func path(name string) (string, error) {
	key := filepath.Clean(name)
	if strings.HasPrefix(key, "../") {
		return "", fmt.Errorf("key outside root: %s", name)
	}
	return key, nil
}

func (fs *VFS) attributes(ctx context.Context, name string) (*Attributes, error) {
	key, err := path(name)
	if err != nil {
		return nil, err
	}

	fs.Lock()
	cachedAttrs, ok := fs.cache.Get(lru.Key(key))
	fs.Unlock()
	if ok {
		return cachedAttrs.(*Attributes), nil
	}

	attrs, err := fs.getAttributes(ctx, key)
	if err != nil {
		return nil, err
	}

	fs.Lock()
	fs.cache.Add(lru.Key(key), attrs)
	fs.Unlock()

	return attrs, nil
}

type notFoundError struct{ key string }

func (nfe notFoundError) Error() string { return fmt.Sprintf("not found: %s", nfe.key) }

func (fs *VFS) getAttributes(ctx context.Context, name string) (*Attributes, error) {
	key, err := path(name)
	if err != nil {
		return nil, err
	}

	obj, err := fs.findObject(ctx, key)
	if err != nil {
		return nil, err
	}

	if obj.IsDir {
		return &Attributes{
			key:     key,
			mode:    os.ModeDir | 0755,
			modTime: obj.ModTime,
		}, nil
	}

	blobAttrs, err := fs.bucket.Attributes(ctx, key)
	if err != nil {
		return nil, err
	}

	mode, err := strconv.Atoi(blobAttrs.Metadata[fileModeKey])
	if err != nil {
		return nil, fmt.Errorf("parse filemode:%v", err)
	}

	attrs := &Attributes{
		key:           key,
		size:          blobAttrs.Size,
		mode:          os.FileMode(mode),
		modTime:       blobAttrs.ModTime,
		symlinkTarget: blobAttrs.Metadata[symlinkTargetKey],
	}

	return attrs, nil
}

func (fs *VFS) findObject(ctx context.Context, key string) (*blob.ListObject, error) {
	const delimiter = "/"
	iter := fs.bucket.List(&blob.ListOptions{
		Prefix:    key,
		Delimiter: delimiter,
	})

	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			return nil, notFoundError{key}
		} else if err != nil {
			return nil, err
		}

		if obj.Key == key || obj.Key == key+delimiter {
			return obj, nil
		}

		if obj.Key > key+delimiter {
			return nil, notFoundError{key}
		}
	}
}

func (fs *VFS) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.attributes(ctx, name)
}

func (fs *VFS) Readlink(ctx context.Context, name string) (string, error) {
	attrs, err := fs.attributes(ctx, name)
	if err != nil {
		return "", err
	}

	if !attrs.IsSymlink() {
		return "", fmt.Errorf("not a symlink: %s", name)
	}

	return attrs.symlinkTarget, nil
}

func (fs *VFS) Open(ctx context.Context, name string) (vfs.File, error) {
	attrs, err := fs.attributes(ctx, name)
	if err != nil {
		return nil, err
	}

	return newBlobReadSeeker(ctx, fs.bucket, attrs.Name(), attrs.Size()), nil
}

func (fs *VFS) WriteFile(ctx context.Context, name string, contents io.Reader) error {
	key, err := path(name)
	if err != nil {
		return err
	}

	w, err := fs.bucket.NewWriter(ctx, key, &blob.WriterOptions{
		Metadata: map[string]string{fileModeKey: strconv.Itoa(0644)},
	})
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, contents); err != nil {
		return err
	}

	return w.Close()
}

func (fs *VFS) WriteSymlink(ctx context.Context, name string, target string) error {
	key, err := path(name)
	if err != nil {
		return err
	}

	return fs.bucket.WriteAll(ctx, key, nil, &blob.WriterOptions{
		Metadata: map[string]string{
			fileModeKey:      strconv.Itoa(int(os.ModeSymlink | 0777)),
			symlinkTargetKey: target,
		},
	})
}

// We want to be able to plug open files into http.ServeContent. For that
// we need to implement io.ReadSeeker. blobReadSeeker wraps
// gocloud.dev/blob in a way that lets us use it as a ReadSeeker.
type blobReadSeeker struct {
	ctx    context.Context
	bucket *blob.Bucket
	key    string
	size   int64

	r   *blob.Reader
	pos int64
}

func newBlobReadSeeker(ctx context.Context, bucket *blob.Bucket, key string, size int64) *blobReadSeeker {
	return &blobReadSeeker{
		ctx:    ctx,
		bucket: bucket,
		key:    key,
		size:   size,
	}
}

func (brs *blobReadSeeker) Close() error {
	if brs.r != nil {
		err := brs.r.Close()
		brs.r = nil
		return err
	}

	return nil
}

func (brs *blobReadSeeker) Read(p []byte) (int, error) {
	if brs.r == nil {
		r, err := brs.bucket.NewRangeReader(brs.ctx, brs.key, brs.pos, -1, nil)
		if err != nil {
			return 0, err
		}

		brs.r = r
	}

	n, err := brs.r.Read(p)
	brs.pos += int64(n)
	return n, err
}

func (brs *blobReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if brs.r != nil {
		if err := brs.r.Close(); err != nil {
			return 0, err
		}
		brs.r = nil
	}

	switch whence {
	case io.SeekStart:
		brs.pos = offset
	case io.SeekCurrent:
		brs.pos += offset
	case io.SeekEnd:
		brs.pos = brs.size - offset
	}
	if brs.pos < 0 {
		return 0, errors.New("blobReadSeeker: negative seek")
	}

	return brs.pos, nil
}
