package httprange

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

// Resource represents any HTTP resource that can be read by a GET operation.
// It holds the resource's URL and metadata about it.
type Resource struct {
	ETag         string
	LastModified string
	Size         int64

	url atomic.Value
	err atomic.Value

	httpClient *http.Client
}

func (r *Resource) URL() string {
	url, _ := r.url.Load().(string)
	return url
}

func (r *Resource) SetURL(url string) {
	if r.URL() == url {
		// We want to avoid cache lines invalidation
		// on CPU due to value change
		return
	}

	r.url.Store(url)
}

func (r *Resource) Err() error {
	err, _ := r.err.Load().(error)
	return err
}

func (r *Resource) Valid() bool {
	return r.Err() == nil
}

func (r *Resource) setError(err error) {
	r.err.Store(err)
}

func (r *Resource) Request() (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", r.URL(), nil)
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

func NewResource(ctx context.Context, url string, httpClient *http.Client) (*Resource, error) {
	// the `h.URL` is likely pre-signed URL or a file:// scheme that only supports GET requests
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// we fetch a single byte and ensure that range requests is additionally supported
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, 0))

	// body will be closed by discardAndClose
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		io.CopyN(io.Discard, res.Body, 1) // since we want to read a single byte
		res.Body.Close()
	}()

	resource := &Resource{
		ETag:         res.Header.Get("ETag"),
		LastModified: res.Header.Get("Last-Modified"),
		httpClient:   httpClient,
	}

	resource.SetURL(url)

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

	case http.StatusNotFound:
		return nil, ErrNotFound

	default:
		return nil, fmt.Errorf("httprange: new resource %d: %q", res.StatusCode, res.Status)
	}
}
