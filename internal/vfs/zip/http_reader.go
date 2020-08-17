package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

type httpReader struct {
	URL string
	Off int64
	N   int64
	res *http.Response
}

var httpClient = &http.Client{
	// TODO: we need connect timeout
	// The longest time the request can be executed
	Timeout:   30 * time.Minute,
	Transport: httptransport.NewTransportWithMetrics(metrics.ZIPHttpReaderReqDuration, metrics.ZIPHttpReaderReqTotal),
}

func (h *httpReader) ensureRequest() error {
	if h.res != nil {
		return nil
	}

	req, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("%d-%d", h.Off, h.Off+h.N-1))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		// TODO: sanitize URL
		return fmt.Errorf("the %q failed with %d: %q", h.URL, res.StatusCode, res.Status)
	}

	return nil
}

func (h *httpReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if err := h.ensureRequest(); err != nil {
		return 0, err
	}

	return h.res.Body.Read(p)
}

func (h *httpReader) Close() error {
	if h.res != nil {
		// TODO: should we read till end?
		return h.res.Body.Close()
	}
	return nil
}

type httpReadAt struct {
	URL string
}

func (h *httpReadAt) ReadAt(p []byte, off int64) (n int, err error) {
	r := httpReader{URL: h.URL, Off: off, N: int64(len(p))}
	defer r.Close()

	// TODO:
	// Even if ReadAt returns n < len(p), it may use all of p as scratch space during the call.
	// If some data is available but not len(p) bytes, ReadAt blocks until either all the data
	// is available or an error occurs. In this respect ReadAt is different from Read.
	return r.Read(p)
}

func isHTTPArchive(path string) bool {
	return strings.HasPrefix(path, "https://")
}

func httpSize(path string) (int64, error) {
	// the `h.URL` is likely presigned only for GET
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("%d-%d", 0, 0))
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer io.Copy(ioutil.Discard, res.Body)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// TODO: sanitize URL
		return 0, fmt.Errorf("the %q failed with %d: %q", path, res.StatusCode, res.Status)
	}

	return res.ContentLength, nil
}

func openZIPHTTPArchive(url string) (*zip.Reader, io.Closer, error) {
	size, err := httpSize(url)
	if err != nil {
		return nil, nil, err
	}

	r, err := zip.NewReader(&httpReadAt{URL: url}, size)
	return r, nil, err
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
