package main

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

var shouldRun = flag.Bool("run-acceptance-tests", false, "Run the acceptance tests?")
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
	if *shouldRun {
		return
	}

	t.Log("Acceptance tests disabled")
	t.SkipNow()
}

func TestUnknownHostReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners)
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
	teardown := RunPagesProcess(t, *pagesBinary, listeners)
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "project/")

		if assert.NoError(t, err) {
			rsp.Body.Close()
			assert.Equal(t, http.StatusOK, rsp.StatusCode)
		}
	}
}
