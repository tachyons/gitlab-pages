package httprange

import (
	"context"
	"io"
)

// RangedReader for a resource.
// Implements the io.ReaderAt interface that can be used with Go's archive/zip package.
type RangedReader struct {
	Resource     *Resource
	cachedReader *Reader
}

func (rr *RangedReader) cachedRead(buf []byte, off int64) (int, error) {
	_, err := rr.cachedReader.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return io.ReadFull(rr.cachedReader, buf)
}

func (rr *RangedReader) ephemeralRead(buf []byte, offset int64) (n int, err error) {
	// we can use context.Background and rely on the Reader's httpClient timeout for ephemeral reads
	reader := NewReader(context.Background(), rr.Resource, offset, int64(len(buf)))
	defer reader.Close()

	return io.ReadFull(reader, buf)
}

// SectionReader partitions a resource from `offset` with a specified `size`
func (rr *RangedReader) SectionReader(ctx context.Context, offset, size int64) *Reader {
	return NewReader(ctx, rr.Resource, offset, size)
}

// ReadAt reads from cachedReader if exists, otherwise fetches a new Resource first.
// Opens a resource and reads len(buf) bytes from offset into buf.
func (rr *RangedReader) ReadAt(buf []byte, offset int64) (n int, err error) {
	if rr.cachedReader != nil {
		return rr.cachedRead(buf, offset)
	}

	return rr.ephemeralRead(buf, offset)
}

// WithCachedReader creates a Reader and saves it to the RangedReader instance.
// It takes a readFunc that will Seek  the contents from Reader.
func (rr *RangedReader) WithCachedReader(ctx context.Context, readFunc func()) {
	rr.cachedReader = NewReader(ctx, rr.Resource, 0, rr.Resource.Size)

	defer func() {
		rr.cachedReader.Close()
		rr.cachedReader = nil
	}()

	readFunc()
}

// NewRangedReader creates a RangedReader object on a given resource
func NewRangedReader(resource *Resource) *RangedReader {
	return &RangedReader{Resource: resource}
}
