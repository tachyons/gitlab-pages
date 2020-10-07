package zip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	dirPrefix      = "public/"
	maxSymlinkSize = 256

	// DefaultOpenTimeout to request an archive and read its contents the first time
	DefaultOpenTimeout      = 30 * time.Second
	DataOffsetCacheInterval = 60 * time.Second
	ReadLinkCacheInterval   = 60 * time.Second
)

var (
	errNotSymlink  = errors.New("not a symlink")
	errSymlinkSize = errors.New("symlink too long")
)

// zipArchive implements the vfs.Root interface.
// It represents a zip archive saving all its files in memory.
// It holds an httprange.Resource that can be read with httprange.RangedReader in chunks.
type zipArchive struct {
	fs *zipVFS

	path        string
	once        sync.Once
	done        chan struct{}
	openTimeout time.Duration

	namespace string

	resource *httprange.Resource
	reader   *httprange.RangedReader
	archive  *zip.Reader
	err      error

	files map[string]*zip.File
}

func newArchive(fs *zipVFS, path string, openTimeout time.Duration) *zipArchive {
	return &zipArchive{
		fs:          fs,
		path:        path,
		done:        make(chan struct{}),
		files:       make(map[string]*zip.File),
		openTimeout: openTimeout,
		namespace:   strconv.FormatInt(atomic.AddInt64(&fs.archiveCount, 1), 10) + ":",
	}
}

func (a *zipArchive) openArchive(parentCtx context.Context) (err error) {
	// return early if openArchive was done already in a concurrent request
	select {
	case <-a.done:
		return a.err

	default:
	}

	ctx, cancel := context.WithTimeout(parentCtx, a.openTimeout)
	defer cancel()

	a.once.Do(func() {
		// read archive once in its own routine with its own timeout
		// if parentCtx is canceled, readArchive will continue regardless and will be cached in memory
		go a.readArchive()
	})

	// wait for readArchive to be done or return if the parent context is canceled
	select {
	case <-a.done:
		return a.err
	case <-ctx.Done():
		err := ctx.Err()
		switch err {
		case context.Canceled:
			log.WithError(err).Traceln("open zip archive request canceled")
		case context.DeadlineExceeded:
			log.WithError(err).Traceln("open zip archive timed out")
		}

		return err
	}
}

// readArchive creates an httprange.Resource that can read the archive's contents and stores a slice of *zip.Files
// that can be accessed later when calling any of th vfs.VFS operations
func (a *zipArchive) readArchive() {
	defer close(a.done)

	// readArchive with a timeout separate from openArchive's
	ctx, cancel := context.WithTimeout(context.Background(), a.openTimeout)
	defer cancel()

	a.resource, a.err = httprange.NewResource(ctx, a.path)
	if a.err != nil {
		metrics.ZipOpened.WithLabelValues("error").Inc()
		return
	}

	// load all archive files into memory using a cached ranged reader
	a.reader = httprange.NewRangedReader(a.resource)
	a.reader.WithCachedReader(ctx, func() {
		a.archive, a.err = zip.NewReader(a.reader, a.resource.Size)
	})

	if a.archive == nil || a.err != nil {
		metrics.ZipOpened.WithLabelValues("error").Inc()
		return
	}

	// TODO: Improve preprocessing of zip archives https://gitlab.com/gitlab-org/gitlab-pages/-/issues/432
	for _, file := range a.archive.File {
		if !strings.HasPrefix(file.Name, dirPrefix) {
			continue
		}
		a.files[file.Name] = file
	}

	// recycle memory
	a.archive.File = nil

	fileCount := float64(len(a.files))
	metrics.ZipOpened.WithLabelValues("ok").Inc()
	metrics.ZipOpenedEntriesCount.Add(fileCount)
	metrics.ZipArchiveEntriesCached.Add(fileCount)
}

func (a *zipArchive) findFile(name string) *zip.File {
	name = filepath.Join(dirPrefix, name)

	if file := a.files[name]; file != nil {
		return file
	}

	if dir := a.files[name+"/"]; dir != nil {
		return dir
	}

	return nil
}

// Open finds the file by name inside the zipArchive and returns a reader that can be served by the VFS
func (a *zipArchive) Open(ctx context.Context, name string) (vfs.File, error) {
	file := a.findFile(name)
	if file == nil {
		return nil, os.ErrNotExist
	}

	dataOffset, err := a.fs.dataOffsetCache.findOrFetch(a.namespace, name, func() (interface{}, error) {
		return file.DataOffset()
	})
	if err != nil {
		return nil, err
	}

	// only read from dataOffset up to the size of the compressed file
	reader := a.reader.SectionReader(ctx, dataOffset.(int64), int64(file.CompressedSize64))

	switch file.Method {
	case zip.Deflate:
		return newDeflateReader(reader), nil
	case zip.Store:
		return reader, nil
	default:
		return nil, fmt.Errorf("unsupported compression method: %x", file.Method)
	}
}

// Lstat finds the file by name inside the zipArchive and returns its FileInfo
func (a *zipArchive) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	file := a.findFile(name)
	if file == nil {
		return nil, os.ErrNotExist
	}

	return file.FileInfo(), nil
}

// ReadLink finds the file by name inside the zipArchive and returns the contents of the symlink
func (a *zipArchive) Readlink(ctx context.Context, name string) (string, error) {
	file := a.findFile(name)
	if file == nil {
		return "", os.ErrNotExist
	}

	if file.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", errNotSymlink
	}

	symlinkValue, err := a.fs.readlinkCache.findOrFetch(a.namespace, name, func() (interface{}, error) {
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		var link [maxSymlinkSize + 1]byte

		// read up to len(symlink) bytes from the link file
		n, err := io.ReadFull(rc, link[:])
		if err != nil && err != io.ErrUnexpectedEOF {
			// if err == io.ErrUnexpectedEOF the link is smaller than len(symlink) so it's OK to not return it
			return nil, err
		}

		return string(link[:n]), nil
	})
	if err != nil {
		return "", err
	}

	symlink := symlinkValue.(string)

	// return errSymlinkSize if the number of bytes read from the link is too big
	if len(symlink) > maxSymlinkSize {
		return "", errSymlinkSize
	}

	return symlink, nil
}

// onEvicted called by the zipVFS.cache when an archive is removed from the cache
func (a *zipArchive) onEvicted() {
	metrics.ZipArchiveEntriesCached.Sub(float64(len(a.files)))
}
