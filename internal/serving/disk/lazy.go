package disk

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var (
	// Make sure lazyFile is a vfs.File to support vfs.ServeCompressedFile
	// if the file is compressed.
	// This should always be satisfied because root.Open always returns
	// a vfs.File.
	_ vfs.File = &lazyFile{}

	// Make sure lazyFile is a ReadSeeker to support http.ServeContent
	// if the file is not compressed.
	// Note: lazyFile.Seek only works if the underlying root.Open returns
	// a vfs.SeekableFile which is the case if the file is not compressed.
	_ io.ReadSeeker = &lazyFile{}

	// ErrInvalidSeeker is returned if lazyFile.Seek is called and the
	// underlying file is not seekable
	ErrInvalidSeeker = errors.New("file is not seekable")
)

type lazyFile struct {
	f    vfs.File
	err  error
	load func() (vfs.File, error)
}

func lazyOpen(ctx context.Context, root vfs.Root, fullPath string) lazyFile {
	lf := lazyFile{
		load: func() (vfs.File, error) {
			return root.Open(ctx, fullPath)
		},
	}

	return lf
}

func (lf lazyFile) Read(p []byte) (int, error) {
	if lf.f == nil && lf.err == nil {
		lf.f, lf.err = lf.load()
	}

	if lf.err != nil {
		return 0, lf.err
	}

	return lf.f.Read(p)
}

func (lf lazyFile) Close() error {
	if lf.f != nil {
		return lf.f.Close()
	}

	return nil
}

func (lf lazyFile) Seek(offset int64, whence int) (int64, error) {
	if lf.f == nil && lf.err == nil {
		lf.f, lf.err = lf.load()
	}

	if lf.err != nil {
		return 0, lf.err
	}

	if sf, ok := lf.f.(io.ReadSeeker); ok {
		return sf.Seek(offset, whence)
	}

	return 0, fmt.Errorf("unable to seek from %T: %w", lf.f, ErrInvalidSeeker)
}
