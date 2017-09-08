package artifact

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

const (
	baseURL             = "/projects/%s/jobs/%s/artifacts"
	hostPatternTemplate = `(?i)\Aartifact~(\d+)~(\d+)\.%s\z`
	minStatusCode       = 200
	maxStatusCode       = 299
)

// Artifact is a struct that is made up of a url.URL, http.Client, and
// regexp.Regexp that is used to proxy requests where applicable.
type Artifact struct {
	server  string
	client  *http.Client
	pattern *regexp.Regexp
}

// New when provided the arguments defined herein, returns a pointer to an
// Artifact that is used to proxy requests.
func New(s string, timeout int, pagesDomain string) *Artifact {
	return &Artifact{
		server:  s,
		client:  &http.Client{Timeout: time.Second * time.Duration(timeout)},
		pattern: hostPatternGen(pagesDomain),
	}

}

// TryMakeRequest will attempt to proxy a request and write it to the argument
// http.ResponseWriter, ultimately returning a bool that indicates if the
// http.ResponseWriter has been written to in any capacity.
func (a *Artifact) TryMakeRequest(host string, w http.ResponseWriter, r *http.Request) bool {
	if a == nil || a.server == "" {
		return false
	}

	reqURL, ok := a.buildURL(host, r.URL.Path)
	if !ok {
		return false
	}

	resp, err := a.client.Get(reqURL.String())
	if err != nil {
		httperrors.Serve502(w)
		return true
	}

	if resp.StatusCode == http.StatusNotFound {
		httperrors.Serve404(w)
		return true
	}

	if resp.StatusCode == http.StatusInternalServerError {
		httperrors.Serve500(w)
		return true
	}

	// we only cache responses within the 2xx series response codes
	if (resp.StatusCode >= minStatusCode) && (resp.StatusCode <= maxStatusCode) {
		w.Header().Set("Cache-Control", "max-age=3600")
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return true
}

// buildURL returns a pointer to a url.URL for where the request should be
// proxied to. The returned bool will indicate if there is some sort of issue
// with the url while it is being generated.
func (a *Artifact) buildURL(host, path string) (*url.URL, bool) {
	ids := a.pattern.FindAllStringSubmatch(host, -1)
	if len(ids) != 1 || len(ids[0]) != 3 {
		return nil, false
	}

	strippedIds := ids[0][1:3]
	body := fmt.Sprintf(baseURL, strippedIds[0], strippedIds[1])
	ourPath := a.server
	if strings.HasSuffix(ourPath, "/") {
		ourPath = ourPath[0:len(ourPath)-1] + body
	} else {
		ourPath = ourPath + body
	}

	if len(path) == 0 || strings.HasPrefix(path, "/") {
		ourPath = ourPath + path
	} else {
		ourPath = ourPath + "/" + path
	}

	u, err := url.Parse(ourPath)
	if err != nil {
		return nil, false
	}
	return u, true
}

// hostPatternGen returns a pointer to a regexp.Regexp that is made up of
// the constant hostPatternTemplate and the argument which represents the pages domain.
// This is used to ensure that the requested page meets not only the hostPatternTemplate
// requirements, but is suffixed with the proper pagesDomain.
func hostPatternGen(pagesDomain string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(hostPatternTemplate, regexp.QuoteMeta(pagesDomain)))
}
