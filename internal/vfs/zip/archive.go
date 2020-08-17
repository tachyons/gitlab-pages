package zip

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const dirPrefix = "public/"
const maxSymlinkSize = 256

type zipArchive struct {
	path   string
	once   sync.Once
	done   chan struct{}
	zip    *zip.ReadCloser
	files  map[string]*zip.File
	zipErr error
}

func (a *zipArchive) Open(ctx context.Context) error {
	a.once.Do(func() {
		a.zip, a.zipErr = zip.OpenReader(a.path)
		if a.zip != nil {
			a.processZip()
		}
		close(a.done)
	})

	// wait for it to close
	// or exit early
	select {
	case <-a.done:
	case <-ctx.Done():
	}
	return a.zipErr
}

func (a *zipArchive) processZip() {
	for _, file := range a.zip.File {
		if !strings.HasPrefix(file.Name, dirPrefix) {
			continue
		}

		a.files[file.Name] = file
	}

	// recycle memory
	a.zip.File = nil
}

func (a *zipArchive) Close() {
	if a.zip != nil {
		a.zip.Close()
		a.zip = nil
	}
}

func (a *zipArchive) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	file := a.files[name]
	if file == nil {
		return nil, os.ErrNotExist
	}

	return file.FileInfo(), nil
}

func (a *zipArchive) Readlink(ctx context.Context, name string) (string, error) {
	file := a.files[name]
	if file == nil {
		return "", os.ErrNotExist
	}

	if file.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", os.ErrInvalid
	}

	rc, err := file.Open()
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(&io.LimitedReader{R: rc, N: maxSymlinkSize})
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (a *zipArchive) Open(ctx context.Context, name string) (vfs.File, error) {
	file := a.files[name]
	if file == nil {
		return nil, os.ErrNotExist
	}

	rc, err := file.Open()
	// TODO: We can support `io.Seeker` if file would not be compressed
	return rc, err
}

func newArchive(path string) zipArchive {
	return &zipArchive{
		path: path,
		done: make(chan struct{}),
	}
}
