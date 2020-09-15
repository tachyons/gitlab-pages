package zip

import (
	"compress/flate"
	"io"
)

// deflateReader wrapper to support reading compressed files.
// Implements the io.ReadCloser interface.
type deflateReader struct {
	reader      io.ReadCloser
	flateReader io.ReadCloser
}

// Read from flateReader
func (r *deflateReader) Read(p []byte) (n int, err error) {
	return r.flateReader.Read(p)
}

// Close all readers
func (r *deflateReader) Close() error {
	r.reader.Close()
	return r.flateReader.Close()
}

func newDeflateReader(r io.ReadCloser) *deflateReader {
	return &deflateReader{
		reader:      r,
		flateReader: flate.NewReader(r),
	}
}
