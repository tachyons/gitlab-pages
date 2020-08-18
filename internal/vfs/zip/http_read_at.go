package zip

import (
	"errors"
	"io"
)

type httpReadAt struct {
	URL          string
	Size         int64
	cached       bool
	cachedReader *httpReader
}

func (h *httpReadAt) cachedRead(p []byte, off int64) (n int, err error) {
	if off < 0 || off > h.Size {
		return 0, errors.New("outside of bounds")
	}

	if h.cachedReader != nil && (h.cachedReader.Off != off || h.cachedReader.N < int64(len(p))) {
		h.cachedReader.Close()
		h.cachedReader = nil
	}

	if h.cachedReader == nil {
		h.cachedReader = &httpReader{URL: h.URL, Off: off, N: h.Size - off}
	}

	return io.ReadFull(h.cachedReader, p)
}

func (h *httpReadAt) ephemeralRead(p []byte, off int64) (n int, err error) {
	r := httpReader{URL: h.URL, Off: off, N: int64(len(p))}
	defer r.Close()

	return io.ReadFull(&r, p)
}

func (h *httpReadAt) ReadAt(p []byte, off int64) (n int, err error) {
	if h.cached {
		return h.cachedRead(p, off)
	}

	return h.ephemeralRead(p, off)
}

func (h *httpReadAt) withCachedReader(fn func()) {
	h.cached = true

	defer func() {
		if h.cachedReader != nil {
			h.cachedReader.Close()
		}
		h.cached = false
	}()

	fn()
}
