package http_range

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var (
	// ErrRangeRequestsNotSupported is returned by Seek and Read
	// when the remote server does not allow range requests (Accept-Ranges was not set)
	ErrRangeRequestsNotSupported = errors.New("range requests are not supported by the remote server")
	// ErrInvalidRange is returned by Read when trying to read past the end of the file
	ErrInvalidRange = errors.New("invalid range")
	// ErrContentHasChanged is returned by Read when the content has changed since the first request
	ErrContentHasChanged = errors.New("content has changed since first request")
)

type Reader struct {
	R            *Resource
	offset, size int64
	res          *http.Response
}

var httpClient = &http.Client{
	// TODO: we need connect timeout
	// The longest time the request can be executed
	Timeout:   30 * time.Minute,
	Transport: httptransport.NewTransportWithMetrics(metrics.ZIPHttpReaderReqDuration, metrics.ZIPHttpReaderReqTotal),
}

func (h *Reader) ensureRequest() (err error) {
	if h.res != nil {
		return nil
	}

	if h.offset < 0 || h.size < 0 || h.offset+h.size > h.R.Size {
		return ErrInvalidRange
	}

	req, err := http.NewRequest("GET", h.R.URL, nil)
	if err != nil {
		return err
	}

	if h.R.LastModified != "" {
		req.Header.Set("If-Range", h.R.LastModified)
	} else if h.R.Etag != "" {
		req.Header.Set("If-Range", h.R.Etag)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", h.offset, h.offset+h.size-1))

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	// cleanup body on failure to avoid memory leak
	defer func() {
		if err != nil {
			res.Body.Close()
		}
	}()

	switch res.StatusCode {
	case http.StatusOK:
		// some servers return 200 OK for bytes=0-
		if h.offset > 0 || h.R.Etag != "" && h.R.Etag != res.Header.Get("ETag") {
			return ErrContentHasChanged
		}
		break

	case http.StatusPartialContent:
		break

	case http.StatusRequestedRangeNotSatisfiable:
		return ErrRangeRequestsNotSupported

	default:
		return fmt.Errorf("failed with %d: %q", res.StatusCode, res.Status)
	}

	h.res = res
	return nil
}

// WithinRange checks if a given data can be read efficiently
func (h *Reader) WithinRange(offset, n int64) bool {
	return h.offset == offset && n <= h.size
}

// Read reads a data into a given buffer
func (h *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if err := h.ensureRequest(); err != nil {
		return 0, err
	}

	n, err := h.res.Body.Read(p)

	if err == nil || err == io.EOF {
		h.offset += int64(n)
		h.size -= int64(n)
	}
	return n, err
}

// Close closes a requests body
func (h *Reader) Close() error {
	if h.res != nil {
		// TODO: should we read till end?
		return h.res.Body.Close()
	}
	return nil
}

// NewReader creates a Reader object on a given resource for a given range
func NewReader(resource *Resource, offset, n int64) *Reader {
	return &Reader{R: resource, offset: offset, size: n}
}
