package zip

import (
	"fmt"
	"io"
	"net/http"
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

var requests int

func (h *httpReader) ensureRequest(requestedSize int) error {
	if h.res != nil {
		return nil
	}

	req, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return err
	}
	requests++

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", h.Off, h.Off+h.N-1))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	println("HTTP Request", "Range", "off=", h.Off, "n=", h.N, "requestedSize=", requestedSize, "statusCode=", res.StatusCode, "requests=", requests)
	if res.StatusCode != http.StatusPartialContent {
		res.Body.Close()
		// TODO: sanitize URL
		return fmt.Errorf("the %q failed with %d: %q", h.URL, res.StatusCode, res.Status)
	}

	h.res = res
	return nil
}

func (h *httpReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if err := h.ensureRequest(len(p)); err != nil {
		return 0, err
	}

	n, err := h.res.Body.Read(p)

	if err == nil || err == io.EOF {
		h.Off += int64(n)
		h.N -= int64(n)
	}
	return n, err
}

func (h *httpReader) Close() error {
	if h.res != nil {
		// TODO: should we read till end?
		return h.res.Body.Close()
	}
	return nil
}
