package zipfs

import (
	"archive/zip"
	"compress/flate"
	"context"
	"fmt"
	"io"
)

// ZipEntry represents an open entry in a zip archive. Either a regular file or a symlink.
type ZipEntry struct {
	fs *VFS
	*Attributes
	ctx context.Context

	r io.Reader
	c io.Closer
}

// Seek implements enough of io.Seeker to support size lookups but
// nothing more.
func (zipEntry *ZipEntry) Seek(offset int64, whence int) (int64, error) {
	if zipEntry.r != nil {
		return 0, fmt.Errorf("seek after read")
	}

	if offset != 0 {
		return 0, fmt.Errorf("unsupported offset")
	}

	switch whence {
	case io.SeekStart:
		return 0, nil
	case io.SeekEnd:
		return zipEntry.Size, nil
	}

	return 0, fmt.Errorf("unsupported whence")
}

func (zipEntry *ZipEntry) Close() error {
	if zipEntry.c == nil {
		return nil
	}

	return zipEntry.c.Close()
}

func (zipEntry *ZipEntry) Read(p []byte) (int, error) {
	if zipEntry.r == nil {
		if err := zipEntry.open(); err != nil {
			return 0, err
		}
	}

	return zipEntry.r.Read(p)
}

func (zipEntry *ZipEntry) open() error {
	offset, err := zipEntry.fs.offset(zipEntry.Idx)
	if err != nil {
		return err
	}

	zipFile, err := zipEntry.fs.opener(zipEntry.ctx)
	if err != nil {
		return err
	}

	if _, err := zipFile.Seek(offset, io.SeekStart); err != nil {
		zipFile.Close()
		return err
	}

	limitReader := io.LimitReader(zipFile, zipEntry.CompressedSize)
	switch zipEntry.Method {
	case zip.Store:
		zipEntry.r = limitReader
	case zip.Deflate:
		zipEntry.r = flate.NewReader(limitReader)
	default:
		zipFile.Close()
		return fmt.Errorf("invalid zip method")
	}

	zipEntry.c = zipFile
	return nil
}
