package httprange

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Resource represents any HTTP resource that can be read by a GET operation.
// It holds the resource's URL and metadata about it.
type Resource struct {
	url          string
	ETag         string
	LastModified string
	Size         int64
	err          error

	lock sync.RWMutex
}

func (r *Resource) SetURL(url string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.url = url
}

func (r *Resource) Err() error {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.err
}

func (r *Resource) setError(err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.err = err
}

func (r *Resource) Request() (*http.Request, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return nil, err
	}

	if r.ETag != "" {
		req.Header.Set("ETag", r.ETag)
	} else if r.LastModified != "" {
		// Last-Modified should be a fallback mechanism in case ETag is not present
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Last-Modified
		req.Header.Set("If-Range", r.LastModified)
	}

	return req, nil
}

func NewResource(ctx context.Context, url string) (*Resource, error) {
	// the `h.URL` is likely pre-signed URL that only supports GET requests
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	// we fetch a single byte and ensure that range requests is additionally supported
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, 0))

	// nolint: bodyclose
	// body will be closed by discardAndClose
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		io.CopyN(ioutil.Discard, res.Body, 1) // since we want to read a single byte
		res.Body.Close()
	}()

	resource := &Resource{
		url:          url,
		ETag:         res.Header.Get("ETag"),
		LastModified: res.Header.Get("Last-Modified"),
	}

	switch res.StatusCode {
	case http.StatusOK:
		resource.Size = res.ContentLength
		return resource, nil

	case http.StatusPartialContent:
		contentRange := res.Header.Get("Content-Range")
		ranges := strings.SplitN(contentRange, "/", 2)
		if len(ranges) != 2 {
			return nil, fmt.Errorf("invalid `Content-Range`: %q", contentRange)
		}

		resource.Size, err = strconv.ParseInt(ranges[1], 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid `Content-Range`: %q %w", contentRange, err)
		}

		return resource, nil

	case http.StatusRequestedRangeNotSatisfiable:
		return nil, ErrRangeRequestsNotSupported

	default:
		return nil, fmt.Errorf("httprange: new resource %d: %q", res.StatusCode, res.Status)
	}
}
