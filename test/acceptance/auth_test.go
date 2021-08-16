package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWhenAuthIsDisabledPrivateIsNotAccessible(t *testing.T) {
	RunPagesProcessWithStubGitLabServer(t,
		withListeners([]ListenSpec{httpListener}),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
}

func TestWhenAuthIsEnabledPrivateWillRedirectToAuthorize(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpsListener})

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	rsp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)
	require.NoError(t, err)

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Equal(t, 1, len(rsp.Header["Location"]))

	url, err = url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	require.Equal(t, "https", url.Scheme)
	require.Equal(t, "public-gitlab-auth.com", url.Host)
	require.Equal(t, "/oauth/authorize", url.Path)
	require.Equal(t, "clientID", url.Query().Get("client_id"))
	require.Equal(t, "https://projects.gitlab-example.com/auth", url.Query().Get("redirect_uri"))
	require.NotEmpty(t, url.Query().Get("scope"))
	require.NotEmpty(t, url.Query().Get("state"))
}

func TestWhenAuthDeniedWillCauseUnauthorized(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpsListener})

	rsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?error=access_denied")

	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
}
func TestWhenLoginCallbackWithWrongStateShouldFail(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpsListener})

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	// Go to auth page with wrong state will cause failure
	authrsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?code=0&state=0")

	require.NoError(t, err)
	defer authrsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, authrsp.StatusCode)
}

func TestWhenLoginCallbackWithUnencryptedCode(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpsListener})

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetPageFromListenerWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
		url.Query().Get("state"), cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will cause 500 because the code is not encrypted
	require.Equal(t, http.StatusInternalServerError, authrsp.StatusCode)
}

func TestAccessControlUnderCustomDomainStandalone(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpListener})

	tests := map[string]struct {
		domain string
		path   string
	}{
		"private_domain_only": {
			domain: "private.domain.com",
			path:   "",
		},
		"private_domain_with_query": {
			domain: "private.domain.com",
			path:   "?q=test",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.domain, tt.path)
			require.NoError(t, err)
			defer rsp.Body.Close()

			cookie := rsp.Header.Get("Set-Cookie")

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			state := url.Query().Get("state")
			require.Equal(t, "http://"+tt.domain, url.Query().Get("domain"))

			pagesrsp, err := GetRedirectPage(t, httpListener, url.Host, url.Path+"?"+url.RawQuery)
			require.NoError(t, err)
			defer pagesrsp.Body.Close()

			pagescookie := pagesrsp.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code=1&state="+
				state, pagescookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain
			require.Equal(t, tt.domain, url.Host)
			code := url.Query().Get("code")
			require.NotEqual(t, "1", code)

			authrsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code="+code+"&state="+
				state, cookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// Will redirect to the page
			cookie = authrsp.Header.Get("Set-Cookie")
			require.Equal(t, http.StatusFound, authrsp.StatusCode)

			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain
			require.Equal(t, "http://"+tt.domain+"/"+tt.path, url.String())

			// Fetch page in custom domain
			authrsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, tt.path, cookie)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, authrsp.StatusCode)
		})
	}
}

func TestCustomErrorPageWithAuth(t *testing.T) {
	runPagesWithAuth(t, []ListenSpec{httpListener})

	tests := []struct {
		name              string
		domain            string
		path              string
		expectedErrorPage string
	}{
		{
			name:              "private_project_authorized",
			domain:            "group.404.gitlab-example.com",
			path:              "/private_project/unknown",
			expectedErrorPage: "Private custom 404 error page",
		},
		{
			name:   "public_namespace_with_private_unauthorized_project",
			domain: "group.404.gitlab-example.com",
			// /private_unauthorized/config.json resolves project ID to 2000 which will cause a 401 from the mock GitLab testServer
			path:              "/private_unauthorized/unknown",
			expectedErrorPage: "Custom 404 group page",
		},
		{
			name:              "private_namespace_authorized",
			domain:            "group.auth.gitlab-example.com",
			path:              "/unknown",
			expectedErrorPage: "group.auth.gitlab-example.com namespace custom 404",
		},
		{
			name:   "private_namespace_with_private_project_auth_failed",
			domain: "group.auth.gitlab-example.com",
			// project ID is 2005 which causes a 401
			path:              "/private.project.1/unknown",
			expectedErrorPage: "The page you're looking for could not be found.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, tt.domain, tt.path)
			require.NoError(t, err)
			defer rsp.Body.Close()

			cookie := rsp.Header.Get("Set-Cookie")

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			state := url.Query().Get("state")
			require.Equal(t, "http://"+tt.domain, url.Query().Get("domain"))

			pagesrsp, err := GetRedirectPage(t, httpListener, url.Host, url.Path+"?"+url.RawQuery)
			require.NoError(t, err)
			defer pagesrsp.Body.Close()

			pagescookie := pagesrsp.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagescookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain
			require.Equal(t, tt.domain, url.Host)
			// code must have changed since it's encrypted
			code := url.Query().Get("code")
			require.NotEqual(t, "1", code)
			require.Equal(t, state, url.Query().Get("state"))

			// Run auth callback in custom domain
			authrsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code="+code+"&state="+
				state, cookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// Will redirect to the page
			groupCookie := authrsp.Header.Get("Set-Cookie")
			require.Equal(t, http.StatusFound, authrsp.StatusCode)

			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain error page
			require.Equal(t, "http://"+tt.domain+tt.path, url.String())

			// Fetch page in custom domain
			anotherResp, err := GetRedirectPageWithCookie(t, httpListener, tt.domain, tt.path, groupCookie)
			require.NoError(t, err)

			require.Equal(t, http.StatusNotFound, anotherResp.StatusCode)

			page, err := ioutil.ReadAll(anotherResp.Body)
			require.NoError(t, err)
			require.Contains(t, string(page), tt.expectedErrorPage)
		})
	}
}

func TestAccessControlUnderCustomDomainWithHTTPSProxy(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuth(t, *pagesBinary, supportedListeners(), testServer.URL, "https://public-gitlab-auth.com")
	defer teardown()

	rsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com", "/", "", true)
	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := url.Query().Get("state")
	require.Equal(t, url.Query().Get("domain"), "https://private.domain.com")
	pagesrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, url.Host, url.Path+"?"+url.RawQuery, "", true)
	require.NoError(t, err)
	defer pagesrsp.Body.Close()

	pagescookie := pagesrsp.Header.Get("Set-Cookie")

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener,
		"projects.gitlab-example.com", "/auth?code=1&state="+state,
		pagescookie, true)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "private.domain.com", url.Host)
	// code must have changed since it's encrypted
	code := url.Query().Get("code")
	require.NotEqual(t, "1", code)
	require.Equal(t, state, url.Query().Get("state"))

	// Run auth callback in custom domain
	authrsp, err = GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com",
		"/auth?code="+code+"&state="+state, cookie, true)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will redirect to the page
	cookie = authrsp.Header.Get("Set-Cookie")
	require.Equal(t, http.StatusFound, authrsp.StatusCode)

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "https://private.domain.com/", url.String())
	// Fetch page in custom domain
	authrsp, err = GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com", "/",
		cookie, true)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestAccessControlGroupDomain404RedirectsAuth(t *testing.T) {
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, supportedListeners(), "https://internal-gitlab-auth.com", "https://public-gitlab-auth.com")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusFound, rsp.StatusCode)
	// Redirects to the projects under gitlab pages domain for authentication flow
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "projects.gitlab-example.com", url.Host)
	require.Equal(t, "/auth", url.Path)
}
func TestAccessControlProject404DoesNotRedirect(t *testing.T) {
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, supportedListeners(), "https://internal-gitlab-auth.com", "https://public-gitlab-auth.com")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/project/nonexistent/")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func setupTransport(t *testing.T) {
	transport := (TestHTTPSClient.Transport).(*http.Transport)
	defer func(t time.Duration) {
		transport.ResponseHeaderTimeout = t
	}(transport.ResponseHeaderTimeout)
	transport.ResponseHeaderTimeout = 5 * time.Second
}

type runPagesFunc func(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, authServer string) func()

func testAccessControl(t *testing.T, runPages runPagesFunc) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	setupTransport(t)

	keyFile, certFile := CreateHTTPSFixtureFiles(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(keyFile)
		os.Remove(certFile)
	})

	testServer := makeGitLabPagesAccessStub(t)
	testServer.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	testServer.StartTLS()
	defer testServer.Close()

	tests := []struct {
		host         string
		path         string
		status       int
		redirectBack bool
		name         string
	}{
		{
			name:         "project with access",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		{
			name:         "project without access",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		{
			name:         "invalid token test should redirect back",
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		{
			name:         "no project should redirect to login and then return 404",
			host:         "group.auth.gitlab-example.com",
			path:         "/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
		{
			name:         "no project should redirect to login and then return 404",
			host:         "nonexistent.gitlab-example.com",
			path:         "/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		}, // subgroups
		{
			name:         "[subgroup] project with access",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		{
			name:         "[subgroup] project without access",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		{
			name:         "[subgroup] invalid token test should redirect back",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		{
			name:         "[subgroup] no project should redirect to login and then return 404",
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
		{
			name:         "[subgroup] no project should redirect to login and then return 404",
			host:         "nonexistent.gitlab-example.com",
			path:         "/subgroup/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardown := runPages(t, *pagesBinary, []ListenSpec{httpsListener}, "", certFile, testServer.URL)
			defer teardown()

			rsp1, err1 := GetRedirectPage(t, httpsListener, tt.host, tt.path)
			require.NoError(t, err1)
			defer rsp1.Body.Close()

			require.Equal(t, http.StatusFound, rsp1.StatusCode)
			cookie := rsp1.Header.Get("Set-Cookie")

			// Redirects to the projects under gitlab pages domain for authentication flow
			loc1, err := url.Parse(rsp1.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "projects.gitlab-example.com", loc1.Host)
			require.Equal(t, "/auth", loc1.Path)
			state := loc1.Query().Get("state")

			rsp2, err2 := GetRedirectPage(t, httpsListener, loc1.Host, loc1.Path+"?"+loc1.RawQuery)
			require.NoError(t, err2)
			defer rsp2.Body.Close()

			require.Equal(t, http.StatusFound, rsp2.StatusCode)
			pagesDomainCookie := rsp2.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp1, err := GetRedirectPageWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagesDomainCookie)
			require.NoError(t, err)
			defer authrsp1.Body.Close()

			// Will redirect auth callback to correct host
			authLoc, err := url.Parse(authrsp1.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tt.host, authLoc.Host)
			require.Equal(t, "/auth", authLoc.Path)

			// Request auth callback in project domain
			authrsp2, err := GetRedirectPageWithCookie(t, httpsListener, authLoc.Host, authLoc.Path+"?"+authLoc.RawQuery, cookie)
			require.NoError(t, err)

			// server returns the ticket, user will be redirected to the project page
			require.Equal(t, http.StatusFound, authrsp2.StatusCode)
			cookie = authrsp2.Header.Get("Set-Cookie")

			rsp3, err3 := GetRedirectPageWithCookie(t, httpsListener, tt.host, tt.path, cookie)
			require.NoError(t, err3)
			defer rsp3.Body.Close()

			require.Equal(t, tt.status, rsp3.StatusCode)
			require.Equal(t, "", rsp3.Header.Get("Cache-Control"))

			if tt.redirectBack {
				loc3, err := url.Parse(rsp3.Header.Get("Location"))
				require.NoError(t, err)

				require.Equal(t, "https", loc3.Scheme)
				require.Equal(t, tt.host, loc3.Host)
				require.Equal(t, tt.path, loc3.Path)
			}
		})
	}
}

func TestAccessControlWithSSLCertFile(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithGitlabServerWithSSLCertFile)
}

func TestAccessControlWithSSLCertDir(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithGitlabServerWithSSLCertDir)
}

// This proves the fix for https://gitlab.com/gitlab-org/gitlab-pages/-/issues/262
// Read the issue description if any changes to internal/auth/ break this test.
// Related to https://tools.ietf.org/html/rfc6749#section-10.6.
func TestHijackedCode(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuth(t, *pagesBinary, supportedListeners(), testServer.URL, "https://public-gitlab-auth.com")
	defer teardown()

	/****ATTACKER******/
	// get valid cookie for a different private project
	targetDomain := "private.domain.com"
	attackersDomain := "group.auth.gitlab-example.com"
	attackerCookie, attackerState := getValidCookieAndState(t, targetDomain)

	/****TARGET******/
	// fool target to click on modified URL with attacker's domain for redirect with a valid state
	hackedURL := fmt.Sprintf("/auth?domain=http://%s&state=%s", attackersDomain, "irrelevant")
	maliciousResp, err := GetProxyRedirectPageWithCookie(t, proxyListener, "projects.gitlab-example.com", hackedURL, "", true)
	require.NoError(t, err)
	defer maliciousResp.Body.Close()

	pagesCookie := maliciousResp.Header.Get("Set-Cookie")

	/*
	   OAuth flow happens here...
	*/
	maliciousRespURL, err := url.Parse(maliciousResp.Header.Get("Location"))
	require.NoError(t, err)
	maliciousState := maliciousRespURL.Query().Get("state")

	// Go to auth page with correct state and code "obtained" from GitLab
	authrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener,
		"projects.gitlab-example.com", "/auth?code=1&state="+maliciousState,
		pagesCookie, true)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	/****ATTACKER******/
	// Target is redirected to attacker's domain and attacker receives the proper code
	require.Equal(t, http.StatusFound, authrsp.StatusCode, "should redirect to attacker's domain")
	authrspURL, err := url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)
	require.Contains(t, authrspURL.String(), attackersDomain)

	// attacker's got the code
	hijackedCode := authrspURL.Query().Get("code")
	require.NotEmpty(t, hijackedCode)

	// attacker tries to access private pages content
	impersonatingRes, err := GetProxyRedirectPageWithCookie(t, proxyListener, targetDomain,
		"/auth?code="+hijackedCode+"&state="+attackerState, attackerCookie, true)
	require.NoError(t, err)
	defer authrsp.Body.Close()

	require.Equal(t, impersonatingRes.StatusCode, http.StatusInternalServerError, "should fail to decode code")
}

func getValidCookieAndState(t *testing.T, domain string) (string, string) {
	t.Helper()

	// follow flow to get a valid cookie
	// visit https://<domain>/
	rsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, domain, "/", "", true)
	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	redirectURL, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := redirectURL.Query().Get("state")
	require.NotEmpty(t, state)

	return cookie, state
}

func runPagesWithAuth(t *testing.T, listeners []ListenSpec) {
	t.Helper()

	//testServer := makeGitLabPagesAccessStub(t)
	//testServer.Start()
	//t.Cleanup(testServer.Close)

	configFile := defaultConfigFileWith(t,
		//"internal-gitlab-server="+testServer.URL,
		"gitlab-server=https://public-gitlab-auth.com",
		"auth-redirect-uri=https://projects.gitlab-example.com/auth",
	)

	RunPagesProcessWithStubGitLabServer(t,
		withListeners(listeners),
		withArguments([]string{
			"-config=" + configFile,
		}),
	)
}
