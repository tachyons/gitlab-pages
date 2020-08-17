package zipfs

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var _ vfs.VFS = &VFS{}

type OpenZipFile interface {
	io.ReadSeeker
	io.ReaderAt
	io.Closer
}

// Opener is a factory that returns a new open zip file. We need this
// because httprs is not thread safe.
type Opener func(ctx context.Context) (OpenZipFile, error)

type VFS struct {
	opener Opener
	expiry time.Time

	files   []*Attributes
	modTime time.Time
	zipReader
	offsetCache
}

// zipReader is an open zipfile. We keep it around to lazily look up entry data offsets.
type zipReader struct {
	*zip.Reader
	c io.Closer
	sync.Mutex
}

func (zr *zipReader) DataOffset(i int) (int64, error) {
	zr.Lock()
	defer zr.Unlock()
	return zr.Reader.File[i].DataOffset()
}

func (zr *zipReader) Close() error { return zr.c.Close() }

// offsetCache stores zip entry data offsets. We cache these offsets
// because each lookup requires an HTTP request and we can only look up
// one offset at a time with httprs.
type offsetCache struct {
	offsets map[int]int64
	sync.RWMutex
}

func seekerSize(s io.Seeker) (int64, error) {
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	_, err = s.Seek(0, io.SeekStart)
	return size, err
}

func Open(ctx context.Context, opener Opener, expiry time.Time) (*VFS, error) {
	fs := &VFS{
		opener: opener,
		expiry: expiry,
	}

	f, err := fs.opener(ctx)
	if err != nil {
		return nil, err
	}

	size, err := seekerSize(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	// This will do IO: it reads and parses the zip central directory
	fs.zipReader.Reader, err = zip.NewReader(f, size)
	if err != nil {
		f.Close()
		return nil, err
	}
	fs.zipReader.c = f

	if err := fs.readAttributesFromZip(); err != nil {
		f.Close()
		return nil, err
	}

	return fs, nil
}

// offset returns the offset in the zip archive where the data of entry i starts.
func (fs *VFS) offset(i int) (int64, error) {
	fs.offsetCache.RLock()
	offset, ok := fs.offsetCache.offsets[i]
	fs.offsetCache.RUnlock()
	if ok {
		return offset, nil
	}

	offset, err := fs.zipReader.DataOffset(i)
	if err != nil {
		return 0, err
	}

	fs.offsetCache.Lock()
	fs.offsetCache.offsets[i] = offset
	fs.offsetCache.Unlock()

	return offset, nil
}

func (fs *VFS) HasExpired() bool { return time.Now().Add(10 * time.Minute).Before(fs.expiry) }

type FileInfo struct {
	*Attributes
	modTime time.Time
}

func (fi FileInfo) Name() string       { return fi.Attributes.Name }
func (fi FileInfo) Size() int64        { return fi.Attributes.Size }
func (fi FileInfo) Mode() os.FileMode  { return fi.Attributes.FileMode }
func (fi FileInfo) ModTime() time.Time { return fi.modTime }
func (fi FileInfo) IsDir() bool        { return fi.Attributes.FileMode.IsDir() }
func (fi FileInfo) Sys() interface{}   { return nil }

type notFound struct{ name string }

func (nf notFound) Error() string { return fmt.Sprintf("not found: %s", nf.name) }

func (fs *VFS) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	attr, ok := fs.getAttributes(name)
	if !ok {
		return nil, notFound{name}
	}

	return FileInfo{Attributes: attr, modTime: fs.modTime}, nil
}

func (fs *VFS) Readlink(ctx context.Context, name string) (string, error) {
	attr, ok := fs.getAttributes(name)
	if !ok {
		return "", notFound{name}
	}
	if attr.FileMode&os.ModeSymlink == 0 {
		return "", fmt.Errorf("not a symlink: %s", name)
	}

	f := fs.zipEntry(ctx, attr)
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	return string(data), err
}

func (fs *VFS) Open(ctx context.Context, name string) (vfs.File, error) {
	attr, ok := fs.getAttributes(name)
	if !ok {
		return nil, notFound{name}
	}

	return fs.zipEntry(ctx, attr), nil
}

func (fs *VFS) zipEntry(ctx context.Context, attr *Attributes) *ZipEntry {
	return &ZipEntry{
		fs:         fs,
		Attributes: attr,
		ctx:        ctx,
	}
}
