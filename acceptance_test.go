package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var pagesBinary = flag.String("gitlab-pages-binary", "./gitlab-pages", "Path to the gitlab-pages binary")

// TODO: Use TCP port 0 everywhere to avoid conflicts. The binary could output
// the actual port (and type of listener) for us to read in place of the
// hardcoded values below.
var listeners = []ListenSpec{
	{"http", "127.0.0.1", "37000"},
	{"http", "::1", "37000"},
	{"https", "127.0.0.1", "37001"},
	{"https", "::1", "37001"},
	{"proxy", "127.0.0.1", "37002"},
	{"proxy", "::1", "37002"},
}

func skipUnlessEnabled(t *testing.T) {
	if testing.Short() {
		t.Log("Acceptance tests disabled")
		t.SkipNow()
	}

	if _, err := os.Stat(*pagesBinary); os.IsNotExist(err) {
		t.Errorf("Couldn't find gitlab-pages binary at %s", *pagesBinary)
		t.FailNow()
	}
}

func TestUnknownHostReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "invalid.invalid", "")

		if assert.NoError(t, err) {
			rsp.Body.Close()
			assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
		}
	}
}

func TestKnownHostReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "project/")

		if assert.NoError(t, err) {
			rsp.Body.Close()
			assert.Equal(t, http.StatusOK, rsp.StatusCode)
		}
	}
}

func TestPrometheusMetricsCanBeScraped(t *testing.T) {
	skipUnlessEnabled(t)
	listener := []ListenSpec{{"http", "127.0.0.1", "37003"}}
	teardown := RunPagesProcess(t, *pagesBinary, listener, ":42345")
	defer teardown()

	resp, err := http.Get("http://localhost:42345/metrics")

	if assert.NoError(t, err) {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		assert.Contains(t, string(body), "gitlab_pages_http_sessions_active 0")
		assert.Contains(t, string(body), "gitlab_pages_domains_served_total 7")
	}
}
