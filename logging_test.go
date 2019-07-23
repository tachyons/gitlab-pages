package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func testLogWithStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww, log.WithField("system", "http"))
	defer w.Log(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(&w, "with-status")
}

func testLogWithoutStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww, log.WithField("system", "http"))
	defer w.Log(r)
	fmt.Fprint(&w, "no-status")
}

func testLogWithDoubleStatus(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww, log.WithField("system", "http"))
	defer w.Log(r)
	w.WriteHeader(http.StatusOK)
	http.Redirect(&w, r, "/test", 301)
}

func TestExtractLogFieldsHidesQueryStrings(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/foo?token=bar", nil)
	r.Header.Set("Referer", "http://invalid.com/bar?token=baz")

	l := newLoggingResponseWriter(w, log.WithField("system", "http"))

	fields := l.extractLogFields(r)

	assert.Equal(t, fields["uri"], "/foo")
	assert.Equal(t, fields["referer"], "http://invalid.com/bar")
}

func TestLoggingWriter(t *testing.T) {
	assert.HTTPBodyContains(t, testLogWithStatus, "GET", "/test", nil, "with-status")
	assert.HTTPBodyContains(t, testLogWithoutStatus, "GET", "/test", nil, "no-status")
	assert.HTTPSuccess(t, testLogWithDoubleStatus, "GET", "/test", nil)
}
