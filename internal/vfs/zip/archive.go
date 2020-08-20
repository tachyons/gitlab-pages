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

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/zip/http_range"
)

const dirPrefix = "public/"
const maxSymlinkSize = 256

type zipArchive struct {
	path string
	once sync.Once
	done chan struct{}

	resource *http_range.Resource
	reader   *http_range.ReadAtReader
	archive  *zip.Reader
	err      error

	files map[string]*zip.File
}

func (a *zipArchive) openArchive(ctx context.Context) error {
	a.once.Do(a.readArchive)

	// wait for it to close
	// or exit early
	select {
	case <-a.done:
	case <-ctx.Done():
	}
	return a.err
}

func (a *zipArchive) readArchive() {
	a.resource, a.err = http_range.NewResource(context.Background(), a.path)
	if a.err != nil {
		return
	}

	a.reader = http_range.NewReadAt(a.resource)
	a.reader.WithCachedReader(func() {
		a.archive, a.err = zip.NewReader(a.reader, a.resource.Size)
	})

	if a.archive != nil {
		for _, file := range a.archive.File {
			if !strings.HasPrefix(file.Name, dirPrefix) {
				continue
			}
			a.files[file.Name] = file
		}

		// recycle memory
		a.archive.File = nil
	}

	close(a.done)
}

func (a *zipArchive) close() {
	// no-op: everything can be GC recycled
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

func (a *zipArchive) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	file := a.findFile(name)
	if file == nil {
		return nil, os.ErrNotExist
	}

	return file.FileInfo(), nil
}

func (a *zipArchive) Readlink(ctx context.Context, name string) (string, error) {
	file := a.findFile(name)
	if file == nil {
		return "", os.ErrNotExist
	}

	if file.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", errors.New("not a symlink")
	}

	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	symlink := make([]byte, maxSymlinkSize+1)
	_, err = io.ReadFull(rc, symlink)
	if err != nil {
		return "", err
	}
	if len(symlink) > maxSymlinkSize {
		return "", errors.New("symlink too long")
	}

	return string(symlink), nil
}

func (a *zipArchive) Open(ctx context.Context, name string) (vfs.File, error) {
	file := a.findFile(name)
	if file == nil {
		return nil, os.ErrNotExist
	}

	dataOffset, err := file.DataOffset()
	if err != nil {
		return nil, err
	}

	// TODO: We can support `io.Seeker` if file would not be compressed
	var reader vfs.File
	reader = a.reader.SectionReader(dataOffset, int64(file.CompressedSize64))

	switch file.Method {
	case zip.Deflate:
		reader = newDeflateReader(reader)

	case zip.Store:
		// no-op

	default:
		return nil, fmt.Errorf("unsupported compression: %x", file.Method)
	}

	return reader, nil
}

func newArchive(path string) *zipArchive {
	return &zipArchive{
		path:  path,
		done:  make(chan struct{}),
		files: make(map[string]*zip.File),
	}
}
