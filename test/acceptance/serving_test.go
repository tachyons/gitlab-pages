package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnknownHostReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "invalid.invalid", "")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestUnknownProjectReturnsNotFound(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestGroupDomainReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestKnownHostReturns200(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	tests := []struct {
		name string
		host string
		path string
	}{
		{
			name: "lower case",
			host: "group.gitlab-example.com",
			path: "project/",
		},
		{
			name: "capital project",
			host: "group.gitlab-example.com",
			path: "CapitalProject/",
		},
		{
			name: "capital group",
			host: "CapitalGroup.gitlab-example.com",
			path: "project/",
		},
		{
			name: "capital group and project",
			host: "CapitalGroup.gitlab-example.com",
			path: "CapitalProject/",
		},
		{
			name: "subgroup",
			host: "group.gitlab-example.com",
			path: "subgroup/project/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, tt.host, tt.path)

				require.NoError(t, err)
				rsp.Body.Close()
				require.Equal(t, http.StatusOK, rsp.StatusCode)
			}
		})
	}
}

func TestNestedSubgroups(t *testing.T) {
	skipUnlessEnabled(t)

	maxNestedSubgroup := 21

	pagesRoot, err := ioutil.TempDir("", "pages-root")
	require.NoError(t, err)
	defer os.RemoveAll(pagesRoot)

	makeProjectIndex := func(subGroupPath string) {
		projectPath := path.Join(pagesRoot, "nested", subGroupPath, "project", "public")
		require.NoError(t, os.MkdirAll(projectPath, 0755))

		projectIndex := path.Join(projectPath, "index.html")
		require.NoError(t, ioutil.WriteFile(projectIndex, []byte("index"), 0644))
	}
	makeProjectIndex("")

	paths := []string{""}
	for i := 1; i < maxNestedSubgroup*2; i++ {
		subGroupPath := fmt.Sprintf("%ssub%d/", paths[i-1], i)
		paths = append(paths, subGroupPath)

		makeProjectIndex(subGroupPath)
	}

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-pages-root", pagesRoot)
	defer teardown()

	for nestingLevel, path := range paths {
		t.Run(fmt.Sprintf("nested level %d", nestingLevel), func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, "nested.gitlab-example.com", path+"project/")

				require.NoError(t, err)
				rsp.Body.Close()
				if nestingLevel <= maxNestedSubgroup {
					require.Equal(t, http.StatusOK, rsp.StatusCode)
				} else {
					require.Equal(t, http.StatusNotFound, rsp.StatusCode)
				}
			}
		})
	}
}

func TestCustom404(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	tests := []struct {
		host    string
		path    string
		content string
	}{
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.404/not/existing-file",
			content: "Custom 404 project page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.404/",
			content: "Custom 404 project page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "not/existing-file",
			content: "Custom 404 group page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "not-existing-file",
			content: "Custom 404 group page",
		},
		{
			host:    "group.404.gitlab-example.com",
			content: "Custom 404 group page",
		},
		{
			host:    "domain.404.com",
			content: "Custom domain.404 page",
		},
		{
			host:    "group.404.gitlab-example.com",
			path:    "project.no.404/not/existing-file",
			content: "The page you're looking for could not be found.",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s/%s", test.host, test.path), func(t *testing.T) {
			for _, spec := range listeners {
				rsp, err := GetPageFromListener(t, spec, test.host, test.path)

				require.NoError(t, err)
				defer rsp.Body.Close()
				require.Equal(t, http.StatusNotFound, rsp.StatusCode)

				page, err := ioutil.ReadAll(rsp.Body)
				require.NoError(t, err)
				require.Contains(t, string(page), test.content)
			}
		})
	}
}

func TestCORSWhenDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-disable-cross-origin-requests")
	defer teardown()

	for _, spec := range listeners {
		for _, method := range []string{"GET", "OPTIONS"} {
			rsp := doCrossOriginRequest(t, spec, method, method, spec.URL("project/"))

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSAllowsGET(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		for _, method := range []string{"GET", "OPTIONS"} {
			rsp := doCrossOriginRequest(t, spec, method, method, spec.URL("project/"))

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, "*", rsp.Header.Get("Access-Control-Allow-Origin"))
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSForbidsPOST(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp := doCrossOriginRequest(t, spec, "OPTIONS", "POST", spec.URL("project/"))

		require.Equal(t, http.StatusOK, rsp.StatusCode)
		require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
		require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
	}
}

func TestCustomHeaders(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-header", "X-Test1:Testing1", "-header", "X-Test2:Testing2")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:", "project/")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		require.Equal(t, "Testing1", rsp.Header.Get("X-Test1"))
		require.Equal(t, "Testing2", rsp.Header.Get("X-Test2"))
	}
}

func TestKnownHostWithPortReturns200(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:"+spec.Port, "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestHttpToHttpsRedirectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpToHttpsRedirectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-redirect-http=true")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))
	require.Equal(t, "https://group.gitlab-example.com/project/", rsp.Header.Get("Location"))

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyGroupEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.https-only.gitlab-example.com", "project1/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyGroupDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.https-only.gitlab-example.com", "project2/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyProjectEnabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "test.my-domain.com", "/index.html")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusMovedPermanently, rsp.StatusCode)
}

func TestHttpsOnlyProjectDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "test2.my-domain.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestHttpsOnlyDomainDisabled(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "no.cert.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestDomainsSource(t *testing.T) {
	skipUnlessEnabled(t)

	type args struct {
		configSource string
		domain       string
		urlSuffix    string
		readyCount   int
	}
	type want struct {
		statusCode int
		content    string
		apiCalled  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "gitlab_source_domain_exists",
			args: args{
				configSource: "gitlab",
				domain:       "new-source-test.gitlab.io",
				urlSuffix:    "/my/pages/project/",
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "New Pages GitLab Source TEST OK\n",
				apiCalled:  true,
			},
		},
		{
			name: "gitlab_source_domain_does_not_exist",
			args: args{
				configSource: "gitlab",
				domain:       "non-existent-domain.gitlab.io",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  true,
			},
		},
		{
			name: "disk_source_domain_exists",
			args: args{
				configSource: "disk",
				// test.domain.com sourced from disk configuration
				domain:    "test.domain.com",
				urlSuffix: "/",
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "main-dir\n",
				apiCalled:  false,
			},
		},
		{
			name: "disk_source_domain_does_not_exist",
			args: args{
				configSource: "disk",
				domain:       "non-existent-domain.gitlab.io",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  false,
			},
		},
		{
			name: "disk_source_domain_should_not_exist_under_hashed_dir",
			args: args{
				configSource: "disk",
				domain:       "hashed.com",
			},
			want: want{
				statusCode: http.StatusNotFound,
				apiCalled:  false,
			},
		},
		{
			name: "auto_source_gitlab_is_not_ready",
			args: args{
				configSource: "auto",
				domain:       "test.domain.com",
				urlSuffix:    "/",
				readyCount:   100, // big number to ensure the API is in bad state for a while
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "main-dir\n",
				apiCalled:  false,
			},
		},
		{
			name: "auto_source_gitlab_is_ready",
			args: args{
				configSource: "auto",
				domain:       "new-source-test.gitlab.io",
				urlSuffix:    "/my/pages/project/",
				readyCount:   0,
			},
			want: want{
				statusCode: http.StatusOK,
				content:    "New Pages GitLab Source TEST OK\n",
				apiCalled:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiCalled bool
			source := NewGitlabDomainsSourceStub(t, &apiCalled, tt.args.readyCount)
			defer source.Close()

			gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

			pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", tt.args.configSource}
			teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, listeners, "", []string{}, pagesArgs...)
			defer teardown()

			response, err := GetPageFromListener(t, httpListener, tt.args.domain, tt.args.urlSuffix)
			require.NoError(t, err)

			require.Equal(t, tt.want.statusCode, response.StatusCode)
			if tt.want.statusCode == http.StatusOK {
				defer response.Body.Close()
				body, err := ioutil.ReadAll(response.Body)
				require.NoError(t, err)

				require.Equal(t, tt.want.content, string(body), "content mismatch")
			}

			require.Equal(t, tt.want.apiCalled, apiCalled, "api called mismatch")
		})
	}
}

func TestKnownHostInReverseProxySetupReturns200(t *testing.T) {
	skipUnlessEnabled(t)

	var listeners = []ListenSpec{
		{"proxy", "127.0.0.1", "37002"},
		{"proxy", "::1", "37002"},
	}

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	for _, spec := range listeners {
		rsp, err := GetProxiedPageFromListener(t, spec, "localhost", "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func doCrossOriginRequest(t *testing.T, spec ListenSpec, method, reqMethod, url string) *http.Response {
	req, err := http.NewRequest(method, url, nil)
	require.NoError(t, err)

	req.Host = "group.gitlab-example.com"
	req.Header.Add("Origin", "example.com")
	req.Header.Add("Access-Control-Request-Method", reqMethod)

	var rsp *http.Response
	err = fmt.Errorf("no request was made")
	for start := time.Now(); time.Since(start) < 1*time.Second; {
		rsp, err = DoPagesRequest(t, spec, req)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, err)

	rsp.Body.Close()
	return rsp
}