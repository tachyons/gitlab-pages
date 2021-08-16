package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnknownHostReturnsNotFound(t *testing.T) {
	RunPagesProcess(t)

	for _, spec := range supportedListeners() {
		rsp, err := GetPageFromListener(t, spec, "invalid.invalid", "")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusNotFound, rsp.StatusCode)
	}
}

func TestUnknownProjectReturnsNotFound(t *testing.T) {
	RunPagesProcess(t)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestGroupDomainReturns200(t *testing.T) {
	RunPagesProcess(t)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)

	body, err := ioutil.ReadAll(rsp.Body)
	require.NoError(t, err)

	require.Equal(t, string(body), "OK\n")
}

func TestKnownHostReturns200(t *testing.T) {
	RunPagesProcess(t)

	tests := []struct {
		name    string
		host    string
		path    string
		content string
	}{
		{
			name:    "lower case",
			host:    "group.gitlab-example.com",
			path:    "project/",
			content: "project-subdir\n",
		},
		{
			name:    "capital project",
			host:    "group.gitlab-example.com",
			path:    "CapitalProject/",
			content: "Capital Project\n",
		},
		{
			name:    "capital group",
			host:    "CapitalGroup.gitlab-example.com",
			path:    "project/",
			content: "Capital Group\n",
		},
		{
			name:    "capital group and project",
			host:    "CapitalGroup.gitlab-example.com",
			path:    "CapitalProject/",
			content: "Capital Group & Project\n",
		},
		{
			name:    "subgroup",
			host:    "group.gitlab-example.com",
			path:    "subgroup/project/",
			content: "A subgroup project\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, spec := range supportedListeners() {
				rsp, err := GetPageFromListener(t, spec, tt.host, tt.path)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, rsp.StatusCode)

				body, err := ioutil.ReadAll(rsp.Body)
				require.NoError(t, err)
				require.Equal(t, tt.content, string(body))

				rsp.Body.Close()
			}
		})
	}
}

func TestCustom404(t *testing.T) {
	RunPagesProcess(t)

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
			for _, spec := range supportedListeners() {
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
	RunPagesProcess(t, withExtraArgument("disable-cross-origin-requests", "true"))

	for _, spec := range supportedListeners() {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
			rsp := doCrossOriginRequest(t, spec, method, method, spec.URL("project/"))

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Origin"))
			require.Equal(t, "", rsp.Header.Get("Access-Control-Allow-Credentials"))
		}
	}
}

func TestCORSAllowsMethod(t *testing.T) {
	RunPagesProcess(t)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedOrigin string
	}{
		{
			name:           "cors-allows-get",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedOrigin: "*",
		},
		{
			name:           "cors-allows-options",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectedOrigin: "*",
		},
		{
			name:           "cors-allows-head",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
			expectedOrigin: "*",
		},
		{
			name:           "cors-forbids-post",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, spec := range supportedListeners() {
				rsp := doCrossOriginRequest(t, spec, tt.method, tt.method, spec.URL("project/"))

				require.Equal(t, tt.expectedStatus, rsp.StatusCode)
				require.Equal(t, tt.expectedOrigin, rsp.Header.Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCustomHeaders(t *testing.T) {
	RunPagesProcess(t,
		withExtraArgument("header", "X-Test1:Testing1"),
		withExtraArgument("header", "X-Test2:Testing2"),
	)

	for _, spec := range supportedListeners() {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:", "project/")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		require.Equal(t, "Testing1", rsp.Header.Get("X-Test1"))
		require.Equal(t, "Testing2", rsp.Header.Get("X-Test2"))
	}
}

func TestKnownHostWithPortReturns200(t *testing.T) {
	RunPagesProcess(t)

	for _, spec := range supportedListeners() {
		rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com:"+spec.Port, "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestHttpToHttpsRedirectDisabled(t *testing.T) {
	RunPagesProcess(t)

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
	RunPagesProcess(t, withExtraArgument("redirect-http", "true"))

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

func TestHTTPSRedirect(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	tests := map[string]struct {
		domain         string
		path           string
		expectedStatus int
	}{
		"domain_https_only_true": {
			domain:         "group.https-only.gitlab-example.com",
			path:           "project1/",
			expectedStatus: http.StatusMovedPermanently,
		},
		"domain_https_only_false": {
			domain:         "group.https-only.gitlab-example.com",
			path:           "project2/",
			expectedStatus: http.StatusOK,
		},
		"custom_domain_https_enabled": {
			domain:         "test.my-domain.com",
			path:           "/index.html",
			expectedStatus: http.StatusMovedPermanently,
		},
		"custom_domain_https_disabled": {
			domain:         "test2.my-domain.com",
			path:           "/",
			expectedStatus: http.StatusOK,
		},
		"custom_domain_https_enabled_with_bad_certificate_is_still_redirected": {
			domain:         "no.cert.com",
			path:           "/",
			expectedStatus: http.StatusMovedPermanently,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// see testdata/api_responses.go for per prefix configuration response from the API
			rsp, err := GetRedirectPage(t, httpListener, test.domain, test.path)
			require.NoError(t, err)

			t.Cleanup(func() {
				rsp.Body.Close()
			})

			require.Equal(t, test.expectedStatus, rsp.StatusCode)
		})
	}
}

// TODO: remove with domain-source-config https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
func TestDomainsSource(t *testing.T) {
	type args struct {
		configSource string
		domain       string
		urlSuffix    string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &stubOpts{
				apiCalled: false,
			}

			source := NewGitlabDomainsSourceStub(t, opts)
			defer source.Close()

			gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)

			pagesArgs := []string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey}
			teardown := RunPagesProcessWithEnvs(t, true, *pagesBinary, []ListenSpec{httpListener}, "", []string{}, pagesArgs...)
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

			require.Equal(t, tt.want.apiCalled, opts.getAPICalled(), "api called mismatch")
		})
	}
}

func TestKnownHostInReverseProxySetupReturns200(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{proxyListener}),
	)

	rsp, err := GetProxiedPageFromListener(t, proxyListener, "127.0.0.1", "group.gitlab-example.com", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestDomainResolverError(t *testing.T) {
	domainName := "new-source-test.gitlab.io"
	opts := &stubOpts{
		apiCalled: false,
	}

	tests := map[string]struct {
		status  int
		panic   bool
		timeout time.Duration
	}{
		"internal_server_errror": {
			status: http.StatusInternalServerError,
		},
		"timeout": {
			timeout: 100 * time.Millisecond,
		},
		"server_fails": {
			panic: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// handler setup
			opts.pagesHandler = func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("host") != domainName {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				opts.setAPICalled(true)
				if test.panic {
					panic("server failed")
				}

				time.Sleep(2 * test.timeout)
				w.WriteHeader(test.status)
			}

			var pagesArgs []string
			if test.timeout != 0 {
				pagesArgs = append(pagesArgs, "-gitlab-client-http-timeout", test.timeout.String(),
					"-gitlab-retrieval-timeout", "200ms", "-gitlab-retrieval-interval", "200ms", "-gitlab-retrieval-retries", "1")
			}

			RunPagesProcess(t,
				withListeners([]ListenSpec{httpListener}),
				withStubOptions(opts),
				withArguments(pagesArgs),
			)

			response, err := GetPageFromListener(t, httpListener, domainName, "/my/pages/project/")
			require.NoError(t, err)
			defer response.Body.Close()

			require.True(t, opts.getAPICalled(), "api must have been called")

			require.Equal(t, http.StatusBadGateway, response.StatusCode)

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), "Something went wrong (502)", "content mismatch")
		})
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

func TestQueryStringPersistedInSlashRewrite(t *testing.T) {
	RunPagesProcess(t)

	rsp, err := GetRedirectPage(t, httpsListener, "group.gitlab-example.com", "project?q=test")
	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))
	require.Equal(t, "//group.gitlab-example.com/project/?q=test", rsp.Header.Get("Location"))

	rsp, err = GetPageFromListener(t, httpsListener, "group.gitlab-example.com", "project/?q=test")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestServerRepliesWithHeaders(t *testing.T) {
	tests := map[string]struct {
		flags           []string
		expectedHeaders map[string][]string
	}{
		"single_header": {
			flags:           []string{"X-testing-1: y-value"},
			expectedHeaders: http.Header{"X-testing-1": {"y-value"}},
		},
		"multiple_header": {
			flags:           []string{"X: 1,2", "Y: 3,4"},
			expectedHeaders: http.Header{"X": {"1,2"}, "Y": {"3,4"}},
		},
	}

	for name, test := range tests {
		testFn := func(envArgs, headerArgs []string) func(*testing.T) {
			return func(t *testing.T) {
				RunPagesProcess(t,
					withListeners([]ListenSpec{httpListener}),
					withEnv(envArgs),
					withArguments(headerArgs),
				)

				rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "/")
				require.NoError(t, err)
				defer rsp.Body.Close()

				require.Equal(t, http.StatusOK, rsp.StatusCode)

				for key, value := range test.expectedHeaders {
					got := rsp.Header.Values(key)
					require.Equal(t, value, got)
				}
			}
		}

		t.Run(name+"/from_single_flag", func(t *testing.T) {
			args := []string{"-header", strings.Join(test.flags, ";;")}
			testFn([]string{}, args)
		})

		t.Run(name+"/from_multiple_flags", func(t *testing.T) {
			args := make([]string, 0, 2*len(test.flags))
			for _, arg := range test.flags {
				args = append(args, "-header", arg)
			}

			testFn([]string{}, args)
		})

		t.Run(name+"/from_config_file", func(t *testing.T) {
			file := newConfigFile(t, "-header="+strings.Join(test.flags, ";;"))

			testFn([]string{}, []string{"-config", file})
		})

		t.Run(name+"/from_env", func(t *testing.T) {
			args := []string{"header", strings.Join(test.flags, ";;")}
			testFn(args, []string{})
		})
	}
}

func TestDiskDisabledFailsToServeFileAndLocalContent(t *testing.T) {
	logBuf := RunPagesProcess(t,
		withExtraArgument("enable-disk", "false"),
	)

	for host, suffix := range map[string]string{
		// API serves "source": { "type": "local" }
		"new-source-test.gitlab.io": "/my/pages/project/",
		// API serves  "source": { "type": "local", "path": "file://..." }
		"zip-from-disk.gitlab.io": "/",
	} {
		t.Run(host, func(t *testing.T) {
			rsp, err := GetPageFromListener(t, httpListener, host, suffix)
			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
		})

		// give the process enough time to write the log message
		require.Eventually(t, func() bool {
			require.Contains(t, logBuf.String(), "cannot serve from disk", "log mismatch")
			return true
		}, time.Second, 10*time.Millisecond)
	}
}
