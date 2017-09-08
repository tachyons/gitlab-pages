package artifact

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTryMakeRequest(t *testing.T) {
	content := "<!DOCTYPE html><html><head><title>Title of the document</title></head><body></body></html>"
	contentType := "text/html; charset=utf-8"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		switch r.URL.Path {
		case "/projects/1/jobs/2/artifacts/200.html":
			w.WriteHeader(http.StatusOK)
		case "/projects/1/jobs/2/artifacts/max-caching.html":
			w.WriteHeader(http.StatusIMUsed)
		case "/projects/1/jobs/2/artifacts/non-caching.html":
			w.WriteHeader(http.StatusTeapot)
		case "/projects/1/jobs/2/artifacts/500.html":
			w.WriteHeader(http.StatusInternalServerError)
		case "/projects/1/jobs/2/artifacts/404.html":
			w.WriteHeader(http.StatusNotFound)
		}
		fmt.Fprint(w, content)
	}))
	defer testServer.Close()

	cases := []struct {
		Path         string
		Status       int
		Content      string
		Length       string
		CacheControl string
		ContentType  string
		Description  string
	}{
		{
			"/200.html",
			http.StatusOK,
			content,
			"90",
			"max-age=3600",
			"text/html; charset=utf-8",
			"basic successful request",
		},
		{
			"/max-caching.html",
			http.StatusIMUsed,
			content,
			"90",
			"max-age=3600",
			"text/html; charset=utf-8",
			"max caching request",
		},
		{
			"/non-caching.html",
			http.StatusTeapot,
			content,
			"90",
			"",
			"text/html; charset=utf-8",
			"no caching request",
		},
	}

	for _, c := range cases {
		result := httptest.NewRecorder()
		reqURL, err := url.Parse(c.Path)
		assert.NoError(t, err)
		r := &http.Request{URL: reqURL}
		art := &Artifact{
			server:  testServer.URL,
			client:  &http.Client{Timeout: time.Second * time.Duration(1)},
			pattern: regexp.MustCompile(fmt.Sprintf(hostPatternTemplate, "gitlab-example.io")),
		}

		assert.True(t, art.TryMakeRequest("artifact~1~2.gitlab-example.io", result, r))
		assert.Equal(t, c.ContentType, result.Header().Get("Content-Type"))
		assert.Equal(t, c.Length, result.Header().Get("Content-Length"))
		assert.Equal(t, c.CacheControl, result.Header().Get("Cache-Control"))
		assert.Equal(t, c.Content, string(result.Body.Bytes()))
		assert.Equal(t, c.Status, result.Code)
	}
}

func TestBuildURL(t *testing.T) {
	cases := []struct {
		RawServer   string
		Host        string
		Path        string
		Expected    string
		PagesDomain string
		Ok          bool
		Description string
	}{
		{
			"https://gitlab.com/api/v4",
			"artifact~1~2.gitlab.io",
			"/path/to/file.txt",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/path/to/file.txt",
			"gitlab.io",
			true,
			"basic case",
		},
		{
			"https://gitlab.com/api/v4/",
			"artifact~1~2.gitlab.io",
			"/path/to/file.txt",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/path/to/file.txt",
			"gitlab.io",
			true,
			"basic case 2",
		},
		{
			"https://gitlab.com/api/v4",
			"artifact~1~2.gitlab.io",
			"path/to/file.txt",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/path/to/file.txt",
			"gitlab.io",
			true,
			"basic case 3",
		},
		{
			"https://gitlab.com/api/v4/",
			"artifact~1~2.gitlab.io",
			"path/to/file.txt",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/path/to/file.txt",
			"gitlab.io",
			true,
			"basic case 4",
		},
		{
			"https://gitlab.com/api/v4",
			"artifact~1~2.gitlab.io",
			"",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts",
			"gitlab.io",
			true,
			"basic case 5",
		},
		{
			"https://gitlab.com/api/v4/",
			"artifact~1~2.gitlab.io",
			"",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts",
			"gitlab.io",
			true,
			"basic case 6",
		},
		{
			"https://gitlab.com/api/v4",
			"artifact~1~2.gitlab.io",
			"/",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/",
			"gitlab.io",
			true,
			"basic case 7",
		},
		{
			"https://gitlab.com/api/v4/",
			"artifact~1~2.gitlab.io",
			"/",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/",
			"gitlab.io",
			true,
			"basic case 8",
		},
		{
			"https://gitlab.com/api/v4",
			"artifact~100000~200000.gitlab.io",
			"/file.txt",
			"https://gitlab.com/api/v4/projects/100000/jobs/200000/artifacts/file.txt",
			"gitlab.io",
			true,
			"expanded case",
		},
		{
			"https://gitlab.com/api/v4/",
			"artifact~1~2.gitlab.io",
			"/file.txt",
			"https://gitlab.com/api/v4/projects/1/jobs/2/artifacts/file.txt",
			"gitlab.io",
			true,
			"server with tailing slash",
		},
		{
			"https://gitlab.com/api/v4",
			"artifact~A~B.gitlab.io",
			"/index.html",
			"",
			"example.com",
			false,
			"non matching domain and request",
		},
		{
			"",
			"artifact~A~B.gitlab.io",
			"",
			"",
			"",
			false,
			"un-parseable Host",
		},
	}

	for _, c := range cases {
		a := &Artifact{server: c.RawServer, pattern: regexp.MustCompile(fmt.Sprintf(hostPatternTemplate, c.PagesDomain))}
		u, ok := a.buildURL(c.Host, c.Path)
		assert.Equal(t, c.Ok, ok, c.Description)
		if c.Ok {
			assert.Equal(t, c.Expected, u.String(), c.Description)
		}
	}
}

func TestMatchHostGen(t *testing.T) {
	cases := []struct {
		URLHost     string
		PagesDomain string
		Expected    bool
		Description string
	}{
		{
			"artifact~1~2.gitlab.io",
			"gitlab.io",
			true,
			"basic case",
		},
		{
			"ARTIFACT~1~2.gitlab.io",
			"gitlab.io",
			true,
			"capital letters case",
		},
		{
			"ARTIFACT~11234~2908908.gitlab.io",
			"gitlab.io",
			true,
			"additional capital letters case",
		},
		{
			"artifact~10000~20000.gitlab.io",
			"gitlab.io",
			true,
			"expanded case",
		},
		{
			"artifact~86753095555~55550935768.gitlab.io",
			"gitlab.io",
			true,
			"large number case",
		},
		{
			"artifact~one~two.gitlab.io",
			"gitlab.io",
			false,
			"letters rather than numbers",
		},
		{
			"artifact~One111~tWo222.gitlab.io",
			"gitlab.io",
			false,
			"Mixture of alphanumeric",
		},
		{
			"artifact~!@#$%~%$#@!.gitlab.io",
			"gitlab.io",
			false,
			"special characters",
		},
		{
			"artifact~1.gitlab.io",
			"gitlab.io",
			false,
			"not enough ids",
		},
		{
			"artifact~1~2~34444~1~4.gitlab.io",
			"gitlab.io",
			false,
			"too many ids",
		},
		{
			"artifact~1~2.gitlab.io",
			"otherhost.io",
			false,
			"different domain / suffix",
		},
	}

	for _, c := range cases {
		reg := hostPatternGen(c.PagesDomain)
		assert.Equal(t, c.Expected, reg.MatchString(c.URLHost), c.Description)
	}
}
