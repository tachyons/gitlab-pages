package http_range

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Resource struct {
	URL          string
	Etag         string
	LastModified string
	Size         int64
}

func NewResource(ctx context.Context, URL string) (*Resource, error) {
	// the `h.URL` is likely presigned only for GET
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	// we fetch a single byte and ensure that range requests is additionally supported
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, 0))
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()

	resource := &Resource{
		URL:          URL,
		Etag:         res.Header.Get("ETag"),
		LastModified: res.Header.Get("Last-Modified"),
	}

	switch res.StatusCode {
	case http.StatusOK:
		resource.Size = res.ContentLength
		println(resource.URL, resource.Etag, resource.LastModified, resource.Size)
		return resource, nil

	case http.StatusPartialContent:
		contentRange := res.Header.Get("Content-Range")
		ranges := strings.SplitN(contentRange, "/", 2)
		if len(ranges) != 2 {
			return nil, fmt.Errorf("invalid `Content-Range`: %q", contentRange)
		}

		resource.Size, err = strconv.ParseInt(ranges[1], 0, 64)
		if err != nil {
			return nil, err
		}

		return resource, nil

	case http.StatusRequestedRangeNotSatisfiable:
		return nil, ErrRangeRequestsNotSupported

	default:
		return nil, fmt.Errorf("failed with %d: %q", res.StatusCode, res.Status)
	}
}
