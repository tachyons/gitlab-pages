package zip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httprange"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	dirPrefix      = "public/"
	maxSymlinkSize = 256

	// DefaultOpenTimeout to request an archive and read its contents the first time
	DefaultOpenTimeout = 30 * time.Second
)

var (
	errNotSymlink  = errors.New("not a symlink")
	errSymlinkSize = errors.New("symlink too long")
)

// zipArchive implements the vfs.Root interface.
// It represents a zip archive saving all its files int memory.
// It holds an httprange.Resource that can be read with httprange.RangedReader in chunks.
type zipArchive struct {
	path        string
	once        sync.Once
	done        chan struct{}
	openTimeout time.Duration

	resource *httprange.Resource
	reader   *httprange.RangedReader
	archive  *zip.Reader
	err      error

	files map[string]*zip.File
}

func newArchive(path string, timeout time.Duration) *zipArchive {
	return &zipArchive{
		path:        path,
		done:        make(chan struct{}),
		files:       make(map[string]*zip.File),
		openTimeout: timeout,
	}
}

func (a *zipArchive) openArchive(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, a.openTimeout)
	defer cancel()

	a.once.Do(func() {
		go a.readArchive(ctx)
	})

	// wait for readArchive to be done or return when the context is canceled
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

func (a *zipArchive) readArchive(ctx context.Context) {
	defer close(a.done)

	a.resource, a.err = httprange.NewResource(ctx, a.path)
	if a.err != nil {
		return
	}

	// load all archive files into memory using a cached ranged reader
	a.reader = httprange.NewRangedReader(a.resource)
	a.reader.WithCachedReader(func() {
		a.archive, a.err = zip.NewReader(a.reader, a.resource.Size)
	})

	if a.archive == nil {
		return
	}

	for _, file := range a.archive.File {
		if !strings.HasPrefix(file.Name, dirPrefix) {
			continue
		}
		a.files[file.Name] = file
	}

	// recycle memory
	a.archive.File = nil
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

	dataOffset, err := file.DataOffset()
	if err != nil {
		return nil, err
	}

	// only read from dataOffset up to the size of the compressed file
	reader := a.reader.SectionReader(dataOffset, int64(file.CompressedSize64))

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

	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	symlink := make([]byte, maxSymlinkSize+1)

	// read up to len(symlink) bytes from the link file
	n, err := io.ReadFull(rc, symlink)
	if err != nil && err != io.ErrUnexpectedEOF {
		// if err == io.ErrUnexpectedEOF the link is smaller than len(symlink) so it's OK to not return it
		return "", err
	}

	// return errSymlinkSize if the number of bytes read from the link is too big
	if n > maxSymlinkSize {
		return "", errSymlinkSize
	}

	// only return the n bytes read from the link
	return string(symlink[:n]), nil
}

// close no-op: everything can be recycled by the GC
func (a *zipArchive) close() {}
