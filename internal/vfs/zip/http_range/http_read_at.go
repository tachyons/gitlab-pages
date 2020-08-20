package http_range

import (
	"io"
)

type ReadAtReader struct {
	R            *Resource
	cachedReader *Reader
}

func (h *ReadAtReader) cachedRead(p []byte, off int64) (n int, err error) {
	if !h.cachedReader.WithinRange(off, int64(len(p))) {
		h.cachedReader.Close()
		h.cachedReader = NewReader(h.R, off, h.R.Size-off)
	}

	return io.ReadFull(h.cachedReader, p)
}

func (h *ReadAtReader) ephemeralRead(p []byte, off int64) (n int, err error) {
	reader := NewReader(h.R, off, int64(len(p)))
	defer reader.Close()

	return io.ReadFull(reader, p)
}

func (h *ReadAtReader) SectionReader(off, n int64) *Reader {
	return NewReader(h.R, off, n)
}

func (h *ReadAtReader) ReadAt(p []byte, off int64) (n int, err error) {
	if h.cachedReader != nil {
		return h.cachedRead(p, off)
	}

	return h.ephemeralRead(p, off)
}

func (h *ReadAtReader) WithCachedReader(fn func()) {
	h.cachedReader = NewReader(h.R, 0, h.R.Size)

	defer func() {
		h.cachedReader.Close()
		h.cachedReader = nil
	}()

	fn()
}

// NewReadAt creates a ReadAt object on a given resource
func NewReadAt(resource *Resource) *ReadAtReader {
	return &ReadAtReader{R: resource}
}
