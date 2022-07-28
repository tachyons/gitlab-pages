package artifact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

const (
	// Format a non-/-suffixed URL, an escaped project full_path, a job ID and
	// a /-prefixed file path into an URL string
	apiURLTemplate = "%s/projects/%s/jobs/%s/artifacts%s"

	minStatusCode = 200
	maxStatusCode = 299

	createArtifactRequestErrMsg = "failed to create the artifact request"
	artifactRequestErrMsg       = "failed to request the artifact"
)

var (
	// Captures subgroup + project, job ID and artifacts path
	pathExtractor       = regexp.MustCompile(`(?i)\A/-/(.*)/-/jobs/(\d+)/artifacts(/[^?]*)\z`)
	errArtifactResponse = errors.New("artifact request response was not successful")
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
			Transport: httptransport.DefaultTransport,
		},
	}
}

// TryMakeRequest will attempt to proxy a request and write it to the argument
// http.ResponseWriter, ultimately returning a bool that indicates if the
// http.ResponseWriter has been written to in any capacity. Additional handler func
// may be given which should return true if it did handle the response.
func (a *Artifact) TryMakeRequest(w http.ResponseWriter, r *http.Request, token string, additionalHandler func(*http.Response) bool) bool {
	if a == nil || a.server == "" {
		return false
	}

	host := request.GetHostWithoutPort(r)

	reqURL, ok := a.BuildURL(host, r.URL.Path)
	if !ok {
		return false
	}

	a.makeRequest(w, r, reqURL, token, additionalHandler)

	return true
}

func (a *Artifact) makeRequest(w http.ResponseWriter, r *http.Request, reqURL *url.URL, token string, additionalHandler func(*http.Response) bool) {
	req, err := http.NewRequestWithContext(r.Context(), "GET", reqURL.String(), nil)
	if err != nil {
		logging.LogRequest(r).WithError(err).Error(createArtifactRequestErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve500(w)
		return
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	// The GitLab API expects this value for Group IP restriction to work properly
	// on requests coming through Pages.
	req.Header.Set("X-Forwarded-For", request.GetRemoteAddrWithoutPort(r))

	resp, err := a.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			httperrors.Serve404(w)
			return
		}

		logging.LogRequest(r).WithError(err).Error(artifactRequestErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve502(w)
		return
	}

	defer resp.Body.Close()

	if additionalHandler(resp) {
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		httperrors.Serve404(w)
		return
	}

	if resp.StatusCode == http.StatusInternalServerError {
		logging.LogRequest(r).Error(errArtifactResponse)
		errortracking.CaptureErrWithReqAndStackTrace(errArtifactResponse, r)
		httperrors.Serve500(w)
		return
	}

	// we only cache responses within the 2xx series response codes and that were not private
	if token == "" {
		addCacheHeader(w, resp)
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func addCacheHeader(w http.ResponseWriter, resp *http.Response) {
	if (resp.StatusCode >= minStatusCode) && (resp.StatusCode <= maxStatusCode) {
		w.Header().Set("Cache-Control", "max-age=3600")
	}
}

// encodePathSegments separately encodes each segment of the path, as
// segments can have special characters in them, if the path is not valid
// and gets re-encoded by URL.Parse, %2f will get replaced with '/',
// breaking the namespace that we pass for group%2fproject.
//
// See https://github.com/golang/go/issues/6658 for more context
func encodePathSegments(path string) string {
	parsed := strings.Split(path, "/")

	var encoded []string
	for _, str := range parsed {
		encoded = append(encoded, url.PathEscape(str))
	}
	return strings.Join(encoded, "/")
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
	artifactPath := encodePathSegments(parts[0][3])

	projectID := url.PathEscape(path.Join(topGroup, restOfPath))
	generated := fmt.Sprintf(apiURLTemplate, a.server, projectID, jobID, artifactPath)

	u, err := url.Parse(generated)
	if err != nil {
		return nil, false
	}
	return u, true
}
