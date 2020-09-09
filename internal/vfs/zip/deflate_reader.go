package zip

import (
	"compress/flate"
	"io"
)

type deflateReader struct {
	R io.ReadCloser
	D io.ReadCloser
}

func (r *deflateReader) Read(p []byte) (n int, err error) {
	return r.D.Read(p)
}

func (r *deflateReader) Close() error {
	r.R.Close()
	return r.D.Close()
}

func newDeflateReader(r io.ReadCloser) *deflateReader {
	return &deflateReader{
		R: r,
		D: flate.NewReader(r),
	}
}
