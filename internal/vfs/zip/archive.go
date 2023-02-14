package zip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	dirPrefix      = "public/"
	maxSymlinkSize = 256
)

var (
	errNotSymlink  = errors.New("not a symlink")
	errSymlinkSize = errors.New("symlink too long")
	errNotFile     = errors.New("not a file")
)

type archiveStatus int

const (
	archiveOpening archiveStatus = iota
	archiveOpenError
	archiveOpened
	archiveCorrupted
)

// zipArchive implements the vfs.Root interface.
// It represents a zip archive saving all its files in memory.
// It holds an httprange.Resource that can be read with httprange.RangedReader in chunks.
type zipArchive struct {
	fs *zipVFS

	once        sync.Once
	done        chan struct{}
	openTimeout time.Duration

	cacheNamespace string

	resource *httprange.Resource
	reader   *httprange.RangedReader
	err      error

	files       map[string]*zip.File
	directories map[string]*zip.FileHeader
}

func newArchive(fs *zipVFS, openTimeout time.Duration) *zipArchive {
	return &zipArchive{
		fs:             fs,
		done:           make(chan struct{}),
		files:          make(map[string]*zip.File),
		directories:    make(map[string]*zip.FileHeader),
		openTimeout:    openTimeout,
		cacheNamespace: strconv.FormatInt(atomic.AddInt64(fs.archiveCount, 1), 10) + ":",
	}
}

func (a *zipArchive) openArchive(parentCtx context.Context, url string) (err error) {
	// always try to update URL on resource
	if a.resource != nil {
		a.resource.SetURL(url)
	}

	// return early if openArchive was done already in a concurrent request
	if status, err := a.openStatus(); status != archiveOpening {
		return err
	}

	ctx, cancel := context.WithTimeout(parentCtx, a.openTimeout)
	defer cancel()

	a.once.Do(func() {
		// read archive once in its own routine with its own timeout
		// if parentCtx is canceled, readArchive will continue regardless and will be cached in memory
		go a.readArchive(url)
	})

	// wait for readArchive to be done or return if the parent context is canceled
	select {
	case <-a.done:
		return a.err
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			log.ContextLogger(parentCtx).WithError(err).Traceln("open zip archive request canceled")
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.ContextLogger(parentCtx).WithError(err).Traceln("open zip archive timed out")
		}

		return err
	}
}

// readArchive creates an httprange.Resource that can read the archive's contents and stores a slice of *zip.Files
// that can be accessed later when calling any of th vfs.VFS operations
func (a *zipArchive) readArchive(url string) {
	defer close(a.done)

	// readArchive with a timeout separate from openArchive's
	ctx, cancel := context.WithTimeout(context.Background(), a.openTimeout)
	defer cancel()

	a.resource, a.err = httprange.NewResource(ctx, url, a.fs.httpClient)
	if a.err != nil {
		log.WithFields(log.Fields{
			"archive_url": url,
		}).WithError(a.err).Infoln("read zip archive request failed")
		metrics.ZipOpened.WithLabelValues("error").Inc()
		return
	}

	var archive *zip.Reader

	// load all archive files into memory using a cached ranged reader
	a.reader = httprange.NewRangedReader(a.resource)
	a.reader.WithCachedReader(ctx, func() {
		archive, a.err = zip.NewReader(a.reader, a.resource.Size)
	})

	if archive == nil || a.err != nil {
		log.WithFields(log.Fields{
			"archive_url": url,
		}).WithError(a.err).Infoln("loading zip archive files into memory failed")
		metrics.ZipOpened.WithLabelValues("error").Inc()
		return
	}

	// TODO: Improve preprocessing of zip archives https://gitlab.com/gitlab-org/gitlab-pages/-/issues/432
	for _, file := range archive.File {
		if !strings.HasPrefix(file.Name, dirPrefix) {
			continue
		}

		// Each Modified timestamp contains a pointer to a unique timezone
		// object. This wastes a lot of memory. By setting the timezone to UTC on
		// each timestamp, we allow the unique timezone objects to be
		// garbage-collected. Also see
		// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/702.
		file.Modified = file.Modified.UTC()

		if file.Mode().IsDir() {
			a.directories[file.Name] = &file.FileHeader
		} else {
			a.files[file.Name] = file
		}

		a.addPathDirectory(file.Name)
	}

	// Each file stores a pointer to the zip.reader.
	// The file slice is not used so we null it out
	// to reduce memory consumption.
	archive.File = nil

	fileCount := float64(len(a.files))
	metrics.ZipOpened.WithLabelValues("ok").Inc()
	metrics.ZipOpenedEntriesCount.Add(fileCount)
	metrics.ZipArchiveEntriesCached.Add(fileCount)
}

// addPathDirectory adds a directory for a given path
func (a *zipArchive) addPathDirectory(pathname string) {
	// Split dir and file from `path`
	pathname, _ = path.Split(pathname)
	if pathname == "" {
		return
	}

	if a.directories[pathname] != nil {
		return
	}

	a.directories[pathname] = &zip.FileHeader{
		Name: pathname,
	}
}

func (a *zipArchive) findFile(name string) *zip.File {
	name = path.Clean(dirPrefix + name)

	return a.files[name]
}

func (a *zipArchive) findDirectory(name string) *zip.FileHeader {
	name = path.Clean(dirPrefix + name)

	return a.directories[name+"/"]
}

// Open finds the file by name inside the zipArchive and returns a reader that can be served by the VFS
func (a *zipArchive) Open(ctx context.Context, name string) (vfs.File, error) {
	file := a.findFile(name)
	if file == nil {
		if a.findDirectory(name) != nil {
			return nil, errNotFile
		}
		return nil, os.ErrNotExist
	}

	if !file.Mode().IsRegular() {
		return nil, errNotFile
	}

	dataOffset, err := a.fs.dataOffsetCache.FindOrFetch(a.cacheNamespace, name, func() (interface{}, error) {
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
	if file != nil {
		return file.FileInfo(), nil
	}

	directory := a.findDirectory(name)
	if directory != nil {
		return directory.FileInfo(), nil
	}

	return nil, os.ErrNotExist
}

// ReadLink finds the file by name inside the zipArchive and returns the contents of the symlink
func (a *zipArchive) Readlink(ctx context.Context, name string) (string, error) {
	file := a.findFile(name)
	if file == nil {
		if a.findDirectory(name) != nil {
			return "", errNotSymlink
		}
		return "", os.ErrNotExist
	}

	if file.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", errNotSymlink
	}

	symlinkValue, err := a.fs.readlinkCache.FindOrFetch(a.cacheNamespace, name, func() (interface{}, error) {
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		var link [maxSymlinkSize + 1]byte

		// read up to len(symlink) bytes from the link file
		n, err := io.ReadFull(rc, link[:])
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
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

func (a *zipArchive) openStatus() (archiveStatus, error) {
	select {
	case <-a.done:
		if a.err != nil {
			return archiveOpenError, a.err
		}

		if a.resource != nil && a.resource.Err() != nil {
			return archiveCorrupted, a.resource.Err()
		}

		return archiveOpened, nil

	default:
		return archiveOpening, nil
	}
}
