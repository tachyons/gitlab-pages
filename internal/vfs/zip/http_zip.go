package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func isHTTPArchive(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func httpSize(path string) (int64, error) {
	// the `h.URL` is likely presigned only for GET
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, 0))
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()

	if res.StatusCode != http.StatusPartialContent {
		// TODO: sanitize URL
		return 0, fmt.Errorf("the %q failed with %d: %q", path, res.StatusCode, res.Status)
	}

	contentRange := res.Header.Get("Content-Range")
	ranges := strings.SplitN(contentRange, "/", 2)
	if len(ranges) != 2 {
		return 0, fmt.Errorf("the %q has invalid `Content-Range`: %q", path, contentRange)
	}

	return strconv.ParseInt(ranges[1], 0, 64)
}

func openZIPHTTPArchive(url string) (zipReader *zip.Reader, closer io.Closer, err error) {
	size, err := httpSize(url)
	if err != nil {
		return nil, nil, err
	}

	httpReader := &httpReadAt{URL: url, Size: size, cached: true}
	httpReader.withCachedReader(func() {
		zipReader, err = zip.NewReader(httpReader, size)
	})

	return zipReader, ioutil.NopCloser(nil), err
}

func openZIPDiskArchive(path string) (*zip.Reader, io.Closer, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	return &r.Reader, r, nil
}

func openZIPArchive(path string) (*zip.Reader, io.Closer, error) {
	if isHTTPArchive(path) {
		return openZIPHTTPArchive(path)
	}

	return openZIPDiskArchive(path)
}
