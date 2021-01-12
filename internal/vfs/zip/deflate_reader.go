package zip

import (
	"bufio"
	"compress/flate"
	"errors"
	"io"
	"sync"
)

var ErrClosedReader = errors.New("deflatereader: reader is closed")

var deflateReaderPool sync.Pool

// deflateReader wrapper to support reading compressed files.
// Implements the io.ReadCloser interface.
type deflateReader struct {
	reader      *bufio.Reader
	closer      io.Closer
	flateReader io.ReadCloser
}

// Read from flateReader
func (r *deflateReader) Read(p []byte) (n int, err error) {
	if r.closer == nil {
		return 0, ErrClosedReader
	}

	return r.flateReader.Read(p)
}

// Close all readers
func (r *deflateReader) Close() error {
	if r.closer == nil {
		return ErrClosedReader
	}

	defer func() {
		r.closer.Close()
		r.closer = nil
		deflateReaderPool.Put(r)
	}()

	return r.flateReader.Close()
}

func (r *deflateReader) reset(rc io.ReadCloser) {
	r.reader.Reset(rc)
	r.closer = rc
	r.flateReader.(flate.Resetter).Reset(r.reader, nil)
}

func newDeflateReader(r io.ReadCloser) *deflateReader {
	if dr, ok := deflateReaderPool.Get().(*deflateReader); ok {
		dr.reset(r)
		return dr
	}

	br := bufio.NewReader(r)

	return &deflateReader{
		reader:      br,
		closer:      r,
		flateReader: flate.NewReader(br),
	}
}
