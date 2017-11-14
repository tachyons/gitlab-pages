package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testLogWithStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(&w, "with-status")
}

func testLogWithoutStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)
	fmt.Fprint(&w, "no-status")
}

func testLogWithDoubleStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)
	w.WriteHeader(http.StatusOK)
	http.Redirect(&w, r, "/test", 301)
}

func TestLoggingWriter(t *testing.T) {
	assert.HTTPBodyContains(t, testLogWithStatus, "GET", "/test", nil, "with-status")
	assert.HTTPBodyContains(t, testLogWithoutStatus, "GET", "/test", nil, "no-status")
	assert.HTTPSuccess(t, testLogWithDoubleStatus, "GET", "/test", nil)
}
