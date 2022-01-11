// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package serving_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/serving"
)

var (
	style   = io.NopCloser(strings.NewReader("p{text-transform: none;}"))
	index   = io.NopCloser(strings.NewReader("<!doctype html><meta charset=utf-8><title>hello</title>"))
	lastMod = time.Now()
)

// nolint: gocyclo // this is vendored code
func TestServeContent(t *testing.T) {
	defer afterTest(t)
	type serveParam struct {
		file        vfs.File
		modtime     time.Time
		contentType string
		etag        string
	}
	servec := make(chan serveParam, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := <-servec
		if p.etag != "" {
			w.Header().Set("ETag", p.etag)
		}
		if p.contentType != "" {
			w.Header().Set("Content-Type", p.contentType)
		}
		serving.ServeCompressedFile(w, r, p.modtime, p.file)
	}))
	defer ts.Close()

	type testCase struct {
		// One of file or content must be set:
		file vfs.File

		modtime          time.Time
		serveETag        string // optional
		serveContentType string // optional
		reqHeader        map[string]string
		wantLastMod      string
		wantContentType  string
		wantContentRange string
		wantStatus       int
	}
	tests := map[string]testCase{
		"no_last_modified": {
			file:             style,
			serveContentType: "text/css; charset=utf-8",
			wantContentType:  "text/css; charset=utf-8",
			wantStatus:       200,
		},
		"with_last_modified": {
			file:             index,
			serveContentType: "text/html; charset=utf-8",
			wantContentType:  "text/html; charset=utf-8",
			modtime:          lastMod,
			wantLastMod:      lastMod.UTC().Format(http.TimeFormat),
			wantStatus:       200,
		},
		"not_modified_modtime": {
			file:      style,
			serveETag: `"foo"`, // Last-Modified sent only when no ETag
			modtime:   lastMod,
			reqHeader: map[string]string{
				"If-Modified-Since": lastMod.UTC().Format(http.TimeFormat),
			},
			wantStatus: 304,
		},
		"not_modified_modtime_with_contenttype": {
			file:             style,
			serveContentType: "text/css", // explicit content type
			serveETag:        `"foo"`,    // Last-Modified sent only when no ETag
			modtime:          lastMod,
			reqHeader: map[string]string{
				"If-Modified-Since": lastMod.UTC().Format(http.TimeFormat),
			},
			wantStatus: 304,
		},
		"not_modified_etag": {
			file:      style,
			serveETag: `"foo"`,
			reqHeader: map[string]string{
				"If-None-Match": `"foo"`,
			},
			wantStatus: 304,
		},
		"if_none_match_mismatch": {
			file:      style,
			serveETag: `"foo"`,
			reqHeader: map[string]string{
				"If-None-Match": `"Foo"`,
			},
			wantStatus:       200,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"if_none_match_malformed": {
			file:      style,
			serveETag: `"foo"`,
			reqHeader: map[string]string{
				"If-None-Match": `,`,
			},
			wantStatus:       200,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"ifmatch_matches": {
			file:      style,
			serveETag: `"A"`,
			reqHeader: map[string]string{
				"If-Match": `"Z", "A"`,
			},
			wantStatus:       200,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"ifmatch_star": {
			file:      style,
			serveETag: `"A"`,
			reqHeader: map[string]string{
				"If-Match": `*`,
			},
			wantStatus:       200,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"ifmatch_failed": {
			file:      style,
			serveETag: `"A"`,
			reqHeader: map[string]string{
				"If-Match": `"B"`,
			},
			wantStatus: 412,
		},
		"ifmatch_fails_on_weak_etag": {
			file:      style,
			serveETag: `W/"A"`,
			reqHeader: map[string]string{
				"If-Match": `W/"A"`,
			},
			wantStatus: 412,
		},
		"if_unmodified_since_true": {
			file:    style,
			modtime: lastMod,
			reqHeader: map[string]string{
				"If-Unmodified-Since": lastMod.UTC().Format(http.TimeFormat),
			},
			wantStatus:       200,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
			wantLastMod:      lastMod.UTC().Format(http.TimeFormat),
		},
		"if_unmodified_since_false": {
			file:    style,
			modtime: lastMod,
			reqHeader: map[string]string{
				"If-Unmodified-Since": lastMod.Add(-2 * time.Second).UTC().Format(http.TimeFormat),
			},
			wantStatus:  412,
			wantLastMod: lastMod.UTC().Format(http.TimeFormat),
		},
		// additional tests
		"missin_content_type": {
			file:            index,
			wantContentType: "text/html; charset=utf-8",
			wantStatus:      http.StatusInternalServerError,
		},
		"if_modified_since_malformed": {
			file:        style,
			modtime:     lastMod,
			wantLastMod: lastMod.UTC().Format(http.TimeFormat),
			reqHeader: map[string]string{
				"If-Modified-Since": "foo",
			},
			wantStatus:       http.StatusOK,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"if_unmodified_since_malformed": {
			file:        style,
			modtime:     lastMod,
			wantLastMod: lastMod.UTC().Format(http.TimeFormat),
			reqHeader: map[string]string{
				"If-Unmodified-Since": "foo",
			},
			wantStatus:       http.StatusOK,
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
		"if_modified_since_true": {
			file:    style,
			modtime: lastMod,
			reqHeader: map[string]string{
				"If-Modified-Since": lastMod.Add(-2 * time.Second).UTC().Format(http.TimeFormat),
			},
			wantStatus:       http.StatusOK,
			wantLastMod:      lastMod.UTC().Format(http.TimeFormat),
			wantContentType:  "text/css; charset=utf-8",
			serveContentType: "text/css; charset=utf-8",
		},
	}
	for testName, tt := range tests {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			servec <- serveParam{
				file:        tt.file,
				modtime:     tt.modtime,
				etag:        tt.serveETag,
				contentType: tt.serveContentType,
			}
			req, err := http.NewRequest(method, ts.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.reqHeader {
				req.Header.Set(k, v)
			}

			c := ts.Client()
			res, err := c.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			if res.StatusCode != tt.wantStatus {
				t.Errorf("test %q using %q: got status = %d; want %d", testName, method, res.StatusCode, tt.wantStatus)
			}
			if g, e := res.Header.Get("Content-Type"), tt.wantContentType; g != e {
				t.Errorf("test %q using %q: got content-type = %q, want %q", testName, method, g, e)
			}
			if g, e := res.Header.Get("Content-Range"), tt.wantContentRange; g != e {
				t.Errorf("test %q using %q: got content-range = %q, want %q", testName, method, g, e)
			}
			if g, e := res.Header.Get("Last-Modified"), tt.wantLastMod; g != e {
				t.Errorf("test %q using %q: got last-modified = %q, want %q", testName, method, g, e)
			}
		}
	}
}

func Test_scanETag(t *testing.T) {
	tests := []struct {
		in         string
		wantETag   string
		wantRemain string
	}{
		{`W/"etag-1"`, `W/"etag-1"`, ""},
		{`"etag-2"`, `"etag-2"`, ""},
		{`"etag-1", "etag-2"`, `"etag-1"`, `, "etag-2"`},
		{"", "", ""},
		{"W/", "", ""},
		{`W/"truc`, "", ""},
		{`w/"case-sensitive"`, "", ""},
		{`"spaced etag"`, "", ""},
	}
	for _, test := range tests {
		etag, remain := serving.ExportScanETag(test.in)
		if etag != test.wantETag || remain != test.wantRemain {
			t.Errorf("scanETag(%q)=%q %q, want %q %q", test.in, etag, remain, test.wantETag, test.wantRemain)
		}
	}
}
