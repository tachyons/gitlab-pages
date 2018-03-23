package artifact

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

const (
	// Format a non-/-suffixed URL, an escaped project full_path, a job ID and
	// a /-prefixed file path into an URL string
	apiURLTemplate = "%s/projects/%s/jobs/%s/artifacts%s"

	minStatusCode = 200
	maxStatusCode = 299
)

var (
	// Captures subgroup + project, job ID and artifacts path
	pathExtractor = regexp.MustCompile(`(?i)\A/-/(.*)/-/jobs/(\d+)/artifacts(/[^?]*)\z`)
)

// Artifact proxies requests for artifact files to the GitLab artifacts API
type Artifact struct {
	server string
	suffix string
	client *http.Client
}

// New when provided the arguments defined herein, returns a pointer to an
// Artifact that is used to proxy requests.
func New(server string, timeoutSeconds int, pagesDomain string) *Artifact {
	return &Artifact{
		server: strings.TrimRight(server, "/"),
		suffix: "." + strings.ToLower(pagesDomain),
		client: &http.Client{
			Timeout:   time.Second * time.Duration(timeoutSeconds),
			Transport: transport,
		},
	}
}

// TryMakeRequest will attempt to proxy a request and write it to the argument
// http.ResponseWriter, ultimately returning a bool that indicates if the
// http.ResponseWriter has been written to in any capacity.
func (a *Artifact) TryMakeRequest(host string, w http.ResponseWriter, r *http.Request) bool {
	if a == nil || a.server == "" || host == "" {
		return false
	}

	reqURL, ok := a.BuildURL(host, r.URL.Path)
	if !ok {
		return false
	}

	a.makeRequest(w, reqURL)
	return true
}

func (a *Artifact) makeRequest(w http.ResponseWriter, reqURL *url.URL) {
	resp, err := a.client.Get(reqURL.String())
	if err != nil {
		httperrors.Serve502(w)
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		httperrors.Serve404(w)
		return
	}

	if resp.StatusCode == http.StatusInternalServerError {
		httperrors.Serve500(w)
		return
	}

	// we only cache responses within the 2xx series response codes
	if (resp.StatusCode >= minStatusCode) && (resp.StatusCode <= maxStatusCode) {
		w.Header().Set("Cache-Control", "max-age=3600")
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return
}

// BuildURL returns a pointer to a url.URL for where the request should be
// proxied to. The returned bool will indicate if there is some sort of issue
// with the url while it is being generated.
//
// The URL is generated from the host (which contains the top-level group and
// ends with the pagesDomain) and the path (which contains any subgroups, the
// project, a job ID and a path
// for the artifact file we want to download)
func (a *Artifact) BuildURL(host, requestPath string) (*url.URL, bool) {
	if !strings.HasSuffix(strings.ToLower(host), a.suffix) {
		return nil, false
	}

	topGroup := host[0 : len(host)-len(a.suffix)]

	parts := pathExtractor.FindAllStringSubmatch(requestPath, 1)
	if len(parts) != 1 || len(parts[0]) != 4 {
		return nil, false
	}

	restOfPath := strings.TrimLeft(strings.TrimRight(parts[0][1], "/"), "/")
	if len(restOfPath) == 0 {
		return nil, false
	}

	jobID := parts[0][2]
	artifactPath := parts[0][3]

	projectID := url.PathEscape(path.Join(topGroup, restOfPath))
	generated := fmt.Sprintf(apiURLTemplate, a.server, projectID, jobID, artifactPath)

	u, err := url.Parse(generated)
	if err != nil {
		return nil, false
	}
	return u, true
}
