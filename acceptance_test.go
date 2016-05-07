package main

import (
	"flag"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TODO: Use TCP port 0 everywhere to avoid conflicts. The binary could output
// the actual port (and type of listener) for us to read in place of the
// hardcoded values below.
type listenSpec struct {
	Type string
	Host string
	Port string
}

func (l listenSpec) URL(suffix string) string {
	scheme := "http"
	if l.Type == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/%s", scheme, l.JoinHostPort(), suffix)
}

// Returns only once the TCP server is open
func (l listenSpec) WaitUntilListening() {
	for {
		conn, _ := net.Dial("tcp", l.JoinHostPort())
		if conn != nil {
			conn.Close()
			break
		}
	}
}

func (l listenSpec) JoinHostPort() string {
	return net.JoinHostPort(l.Host, l.Port)
}

var shouldRun = flag.Bool("run-acceptance-tests", false, "Run the acceptance tests?")
var pagesBinary = flag.String("gitlab-pages-binary", "./gitlab-pages", "Path to the gitlab-pages binary")

var listenHTTP = []listenSpec{
	{"http", "127.0.0.1", "3700"},
	{"http", "::1", "3700"},
}

// TODO: listenHTTPS will require TLS configuration
var listenHTTPS = []listenSpec{}

var listenProxy = []listenSpec{
	{"proxy", "127.0.0.1", "3702"},
	{"proxy", "::1", "37002"},
}

var listeners = append(listenHTTP, append(listenHTTPS, listenProxy...)...)

// TODO: start one pages process for all tests?
func runPages(t *testing.T) *exec.Cmd {
	if !*shouldRun {
		t.Log("Acceptance tests disabled")
		t.SkipNow()
	}

	if _, err := os.Stat(*pagesBinary); err != nil {
		t.Logf("Can't find Gitlab Pages binary (%s): %s", *pagesBinary, err)
		t.FailNow()
	}

	var args []string
	for _, spec := range listeners {
		args = append(args, "-listen-"+spec.Type, spec.JoinHostPort())
	}

	cmd := exec.Command(*pagesBinary, args...)
	t.Logf("Running %s %v", *pagesBinary, args)
	cmd.Start()

	// Wait for all TCP servers to be open. Even with this, gitlab-pages
	// will sometimes return 404 if a HTTP request comes in before it has
	// updated its set of domains. This usually takes < 1ms, hence the sleep
	// for now. Without it, intermittent failures occur.
	//
	// TODO: replace this with explicit status from the pages binary
	// TODO: fix the first-request race
	for _, spec := range listeners {
		spec.WaitUntilListening()
	}
	time.Sleep(50 * time.Millisecond)

	return cmd
}

func stopPages(cmd *exec.Cmd) {
	cmd.Process.Kill()
	cmd.Process.Wait()
}

func getPage(t *testing.T, spec listenSpec, host, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host

	t.Logf("curl -H'Host: %s' %s", host, url)
	return http.DefaultClient.Do(req)
}

func TestUnknownHostReturnsNotFound(t *testing.T) {
	cmd := runPages(t)
	defer stopPages(cmd)

	for _, spec := range listeners {
		rsp, err := getPage(t, spec, "invalid.invalid", "")

		if assert.NoError(t, err) {
			rsp.Body.Close()
			assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
		}
	}
}

func TestKnownHostReturns200(t *testing.T) {
	cmd := runPages(t)
	defer stopPages(cmd)

	for _, spec := range listeners {
		rsp, err := getPage(t, spec, "group.gitlab-example.com", "project/")

		if assert.NoError(t, err) {
			rsp.Body.Close()
			assert.Equal(t, http.StatusOK, rsp.StatusCode)
		}
	}
}
