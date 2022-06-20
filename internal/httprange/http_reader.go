package httprange

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var (
	// ErrNotFound is returned when servers responds with 404
	ErrNotFound = errors.New("resource not found")

	// ErrRangeRequestsNotSupported is returned by Seek and Read
	// when the remote server does not allow range requests for a given request parameters
	ErrRangeRequestsNotSupported = errors.New("requests range is not supported by the remote server")

	// ErrInvalidRange is returned by Read when trying to read past the end of the file
	ErrInvalidRange = errors.New("invalid range")

	// seek errors no need to export them
	errSeekInvalidWhence = errors.New("invalid whence")
	errSeekOutsideRange  = errors.New("outside of range")

	rangeRequestPrepareErrMsg = "failed to prepare HTTP range request"
	rangeRequestFailedErrMsg  = "failed HTTP range response"
)

// Reader holds a Resource and specifies ranges to read from at a time.
// Implements the io.Reader, io.Seeker and io.Closer  interfaces.
type Reader struct {
	// ctx for read requests
	ctx context.Context
	// Resource to read from
	Resource *Resource
	// res defines a current response serving data
	res *http.Response
	// rangeStart defines a starting range
	rangeStart int64
	// rangeSize defines a size of range
	rangeSize int64
	// offset defines a current place where data is being read from
	offset int64
}

// ensure that Reader is seekable
var _ vfs.SeekableFile = &Reader{}

// ensureResponse is set before reading from it.
// It will do the request if the reader hasn't got it yet.
func (r *Reader) ensureResponse() error {
	if r.res != nil {
		return nil
	}

	req, err := r.prepareRequest()
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"range_start":   r.rangeStart,
			"range_size":    r.rangeSize,
			"offset":        r.offset,
			"resource_size": r.Resource.Size,
			"resource_url":  logging.CleanURL(r.Resource.URL()),
		}).Error(rangeRequestPrepareErrMsg)
		return err
	}

	metrics.HTTPRangeOpenRequests.Inc()

	res, err := r.Resource.httpClient.Do(req)
	if err != nil {
		metrics.HTTPRangeOpenRequests.Dec()
		return err
	}

	err = r.setResponse(res)
	if err != nil {
		metrics.HTTPRangeOpenRequests.Dec()

		// cleanup body on failure from r.setResponse to avoid memory leak
		res.Body.Close()
		logging.LogRequest(req).WithError(err).WithFields(log.Fields{
			"range_start":   r.rangeStart,
			"range_size":    r.rangeSize,
			"offset":        r.offset,
			"resource_size": r.Resource.Size,
			"resource_url":  logging.CleanURL(r.Resource.URL()),
			"status":        res.StatusCode,
			"status_text":   res.Status,
		}).Error(rangeRequestFailedErrMsg)
	}

	return err
}

func (r *Reader) prepareRequest() (*http.Request, error) {
	if r.rangeStart < 0 || r.rangeSize < 0 || r.rangeStart+r.rangeSize > r.Resource.Size {
		return nil, ErrInvalidRange
	}

	if r.offset < r.rangeStart || r.offset >= r.rangeStart+r.rangeSize {
		return nil, ErrInvalidRange
	}

	req, err := r.Resource.Request()
	if err != nil {
		return nil, err
	}

	req = req.WithContext(r.ctx)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", r.offset, r.rangeStart+r.rangeSize-1))

	return req, nil
}

func (r *Reader) setResponse(res *http.Response) error {
	// TODO: add metrics https://gitlab.com/gitlab-org/gitlab-pages/-/issues/448
	switch res.StatusCode {
	case http.StatusOK:
		// some servers return 200 OK for bytes=0-
		// TODO: should we handle r.Resource.Last-Modified as well?
		if r.offset > 0 || r.Resource.ETag != "" && r.Resource.ETag != res.Header.Get("ETag") {
			r.Resource.setError(ErrRangeRequestsNotSupported)
			return ErrRangeRequestsNotSupported
		}
	case http.StatusNotFound:
		r.Resource.setError(ErrNotFound)
		return ErrNotFound
	case http.StatusPartialContent:
		// Requested `Range` request succeeded https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/206
		break
	case http.StatusRequestedRangeNotSatisfiable:
		r.Resource.setError(ErrRangeRequestsNotSupported)
		return ErrRangeRequestsNotSupported
	default:
		return fmt.Errorf("httprange: read response %d: %q", res.StatusCode, res.Status)
	}

	r.res = res

	return nil
}

// Seek returns the new offset relative to the start of the file and an error, if any.
// io.SeekStart means relative to the start of the file,
// io.SeekCurrent means relative to the current offset, and
// io.SeekEnd means relative to the end.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = r.rangeStart + offset

	case io.SeekCurrent:
		newOffset = r.offset + offset

	case io.SeekEnd:
		newOffset = r.rangeStart + r.rangeSize + offset

	default:
		return 0, errSeekInvalidWhence
	}

	if newOffset < r.rangeStart || newOffset > r.rangeStart+r.rangeSize {
		return 0, errSeekOutsideRange
	}

	if newOffset != r.offset {
		// recycle r.res
		r.Close()
	}

	r.offset = newOffset
	return newOffset - r.rangeStart, nil
}

// Read data into a given buffer.
func (r *Reader) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	if err := r.ensureResponse(); err != nil {
		return 0, vfs.NewReadError(err)
	}

	n, err := r.res.Body.Read(buf)
	if err == nil || errors.Is(err, io.EOF) {
		r.offset += int64(n)
	}

	return n, err
}

// Close closes a requests body
func (r *Reader) Close() error {
	if r.res != nil {
		// no need to read until the end
		err := r.res.Body.Close()
		r.res = nil

		metrics.HTTPRangeOpenRequests.Dec()

		return err
	}

	return nil
}

// NewReader creates a Reader object on a given resource for a given range
func NewReader(ctx context.Context, resource *Resource, offset, size int64) *Reader {
	return &Reader{ctx: ctx, Resource: resource, rangeStart: offset, rangeSize: size, offset: offset}
}
