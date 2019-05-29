package buildservice

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
)

const (
	proxyURLTemplate = "%s/api/v4/jobs/%s/proxy/%s/%s"
	domainKeyword    = "proxy"
)

// The domain structure is:
// username.domainKeywork.build_id.service_name.service_port.pages.domain
// Ex: root.proxy.1.service_name.service_port.xip.io:3010
// Service_port can be either a number or a string
var (
	// Captures subgroup + project, job ID and artifacts path
	domainExtractor = regexp.MustCompile(`(?i)\A(.+)\.` + domainKeyword + `\.(\d+)\.(.+)\.(.+)\z`)
)

// Artifact proxies requests for artifact files to the GitLab artifacts API
type BuildService struct {
	server string
	suffix string
	client *http.Client
}

// New when provided the arguments defined herein, returns a pointer to an
// BuildService that is used to proxy requests.
func New(server string, timeoutSeconds int, pagesDomain string) *BuildService {
	return &BuildService{
		server: strings.TrimRight(server, "/"),
		suffix: "." + strings.ToLower(pagesDomain),
		client: &http.Client{
			Timeout:   time.Second * time.Duration(timeoutSeconds),
			Transport: httptransport.Transport,
		},
	}
}

// TryMakeRequest will attempt to proxy a request and write it to the argument
// http.ResponseWriter, ultimately returning a bool that indicates if the
// http.ResponseWriter has been written to in any capacity.
func (a *BuildService) TryMakeRequest(host string, token string, w http.ResponseWriter, r *http.Request) bool {
	if a.server == "" || host == "" || token == "" {
		return false
	}

	rewrittenReq, ok := a.rewriteRequest(host, r)
	if !ok {
		return false
	}

	a.makeRequest(w, rewrittenReq, token)
	return true
}

func (a *BuildService) makeRequest(w http.ResponseWriter, req *http.Request, token string) {
	u, _ := url.Parse(a.server)
	proxy := httputil.NewSingleHostReverseProxy(u)

	req.Header.Add("Authorization", "Bearer "+token)
	proxy.ServeHTTP(w, req)
}

// BuildURL returns a pointer to a url.URL for where the request should be
// proxied to. The returned bool will indicate if there is some sort of issue
// with the url while it is being generated.
//
// The URL is generated from the host (which contains the top-level group and
// ends with the pagesDomain) and the path (which contains any subgroups, the
// project, a job ID
func (a *BuildService) rewriteRequest(host string, r *http.Request) (*http.Request, bool) {
	if !strings.HasSuffix(strings.ToLower(host), a.suffix) {
		return nil, false
	}

	topDomain := host[0 : len(host)-len(a.suffix)]
	domainParts := domainExtractor.FindAllStringSubmatch(topDomain, 1)
	if len(domainParts) != 1 || len(domainParts[0]) != 5 {
		return nil, false
	}

	jobID := domainParts[0][2]
	serviceName := domainParts[0][3]
	servicePort := domainParts[0][4]
	generated := fmt.Sprintf(proxyURLTemplate, a.server, jobID, serviceName, servicePort)

	u, err := url.Parse(generated)
	if err != nil {
		return nil, false
	}

	log.Printf("r.URL.RawQuery: %#+v\n", r.URL.RawQuery)
	q := url.Values{"path": []string{r.URL.Path}}
	q["wrapped_raw_query_"+jobID] = []string{r.URL.RawQuery}
	u.RawQuery = q.Encode()

	// Rewritten request
	r.URL = u
	log.Printf("r.URL.Pathj: %#+v\n", r.URL.Path)
	log.Printf("r.URL.Query(): %#+v\n", r.URL.Query())

	// if err := r.ParseForm(); err != nil {
	// 	return nil, false
	// }

	// r.Form.Set("HTTP_JOB_TOKEN", "2z3OV406fjsxgAF3xUjS0JA/+u9sVpr0vKhU0FuLRwquFJx5")

	return r, true
}
