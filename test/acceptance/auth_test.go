package acceptance_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestWhenAuthIsDisabledPrivateIsNotAccessible(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
}

func TestWhenAuthIsEnabledPrivateWillRedirectToAuthorize(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Len(t, rsp.Header["Location"], 1)
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	rsp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	require.Equal(t, http.StatusFound, rsp.StatusCode)
	require.Len(t, rsp.Header["Location"], 1)

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
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?error=access_denied")

	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	require.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
}
func TestWhenLoginCallbackWithWrongStateShouldFail(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	// Go to auth page with wrong state will cause failure
	authrsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?code=0&state=0")

	require.NoError(t, err)
	testhelpers.Close(t, authrsp.Body)

	require.Equal(t, http.StatusUnauthorized, authrsp.StatusCode)
}

func TestWhenLoginCallbackWithUnencryptedCode(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	header := http.Header{
		"Cookie": []string{cookie},
	}

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetPageFromListenerWithHeaders(t, httpsListener, "group.auth.gitlab-example.com", "/auth?code=1&state="+
		url.Query().Get("state"), header)

	require.NoError(t, err)
	testhelpers.Close(t, authrsp.Body)

	// Will cause 500 because the code is not encrypted
	require.Equal(t, http.StatusInternalServerError, authrsp.StatusCode)
}

func TestAccessControlUnderCustomDomain(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

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
			// visit to custom domain
			rsp, err := GetRedirectPage(t, httpListener, tt.domain, tt.path)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			domainCookie := rsp.Header.Get("Set-Cookie")

			projectProxyURL, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			state := projectProxyURL.Query().Get("state")
			require.Equal(t, "http://"+tt.domain, projectProxyURL.Query().Get("domain"))

			// visit projects.gitlab-example.com?state=something
			projectsProxyRsp, err := GetRedirectPage(t, httpListener,
				projectProxyURL.Host, projectProxyURL.Path+"?"+projectProxyURL.RawQuery)
			require.NoError(t, err)
			testhelpers.Close(t, projectsProxyRsp.Body)

			projectsCookie := projectsProxyRsp.Header.Get("Set-Cookie")

			// visit projects.gitlab-example.com?state=something&code=1
			authRsp, err := GetRedirectPageWithCookie(t, httpListener, projectProxyURL.Host, "/auth?code=1&state="+
				state, projectsCookie)

			require.NoError(t, err)
			testhelpers.Close(t, authRsp.Body)

			backDomainURL, err := projectProxyURL.Parse(authRsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain
			require.Equal(t, tt.domain, backDomainURL.Host)
			code := backDomainURL.Query().Get("code")
			require.NotEqual(t, "1", code)

			// visit domain.com/auth?code&state will set the cookie and redirect back to original page
			selfRedirectRsp, err := GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code="+code+"&state="+
				state, domainCookie)

			require.NoError(t, err)
			testhelpers.Close(t, selfRedirectRsp.Body)

			// Will redirect to the page
			domainCookie = selfRedirectRsp.Header.Get("Set-Cookie")
			require.Equal(t, http.StatusFound, selfRedirectRsp.StatusCode)

			selfRedirectURL, err := projectProxyURL.Parse(selfRedirectRsp.Header.Get("Location"))
			require.NoError(t, err)

			// Will redirect to custom domain
			require.Equal(t, "http://"+tt.domain+"/"+tt.path, selfRedirectURL.String())

			// Fetch page in custom domain
			authRsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, tt.path, domainCookie)
			require.NoError(t, err)
			testhelpers.Close(t, authRsp.Body)
			require.Equal(t, http.StatusOK, authRsp.StatusCode)

			// Try to fetch page from another domain
			// it should restart the auth process ignoring already existing cookie
			secondAuthRsp, err := GetRedirectPageWithCookie(t, httpListener, "group.auth.gitlab-example.com", "/private.project/", domainCookie)
			require.NoError(t, err)
			testhelpers.Close(t, authRsp.Body)

			secondAuthURL, err := url.Parse(secondAuthRsp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "projects.gitlab-example.com", secondAuthURL.Host)
			require.Equal(t, "/auth", secondAuthURL.Path)
			require.Equal(t, "http://group.auth.gitlab-example.com", secondAuthURL.Query().Get("domain"))
		})
	}
}

func TestCustomErrorPageWithAuth(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

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
			testhelpers.Close(t, rsp.Body)

			cookie := rsp.Header.Get("Set-Cookie")

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			state := url.Query().Get("state")
			require.Equal(t, "http://"+tt.domain, url.Query().Get("domain"))

			pagesrsp, err := GetRedirectPage(t, httpListener, url.Host, url.Path+"?"+url.RawQuery)
			require.NoError(t, err)
			testhelpers.Close(t, pagesrsp.Body)

			pagescookie := pagesrsp.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagescookie)

			require.NoError(t, err)
			testhelpers.Close(t, authrsp.Body)

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
			testhelpers.Close(t, authrsp.Body)

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
			testhelpers.Close(t, anotherResp.Body)

			require.Equal(t, http.StatusNotFound, anotherResp.StatusCode)

			page, err := io.ReadAll(anotherResp.Body)
			require.NoError(t, err)
			require.Contains(t, string(page), tt.expectedErrorPage)
		})
	}
}

func TestAccessControlUnderCustomDomainWithHTTPSProxy(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{proxyListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com", "/", "", true)
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := url.Query().Get("state")
	require.Equal(t, url.Query().Get("domain"), "https://private.domain.com")
	pagesrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, url.Host, url.Path+"?"+url.RawQuery, "", true)
	require.NoError(t, err)
	testhelpers.Close(t, pagesrsp.Body)

	pagescookie := pagesrsp.Header.Get("Set-Cookie")

	// Go to auth page with correct state will cause fetching the token
	authrsp, err := GetProxyRedirectPageWithCookie(t, proxyListener,
		"projects.gitlab-example.com", "/auth?code=1&state="+state,
		pagescookie, true)

	require.NoError(t, err)
	testhelpers.Close(t, authrsp.Body)

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
	testhelpers.Close(t, authrsp.Body)

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
	testhelpers.Close(t, authrsp.Body)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestAccessControlGroupDomain404RedirectsAuth(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/nonexistent/")
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)
	require.Equal(t, http.StatusFound, rsp.StatusCode)
	// Redirects to the projects under gitlab pages domain for authentication flow
	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "projects.gitlab-example.com", url.Host)
	require.Equal(t, "/auth", url.Path)
}

func TestAccessControlProject404DoesNotRedirect(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

	rsp, err := GetRedirectPage(t, httpListener, "group.gitlab-example.com", "/project/nonexistent/")
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

type runPagesFunc func(t *testing.T, listeners []ListenSpec, sslCertFile string)

func testAccessControl(t *testing.T, runPages runPagesFunc) {
	_, certFile := CreateHTTPSFixtureFiles(t)

	tests := map[string]struct {
		host         string
		path         string
		status       int
		redirectBack bool
	}{
		"project_with_access": {
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		"project_without_access": {
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		"invalid_token_test_should_redirect_back": {
			host:         "group.auth.gitlab-example.com",
			path:         "/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		"no_project_should_redirect_to_login_and_then_return404": {
			host:         "group.auth.gitlab-example.com",
			path:         "/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
		// subgroups
		"subgroup_project_with_access": {
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project/",
			status:       http.StatusOK,
			redirectBack: false,
		},
		"subgroup_project_without_access": {
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.1/",
			status:       http.StatusNotFound, // Do not expose project existed
			redirectBack: false,
		},
		"subgroup_invalid_token_test_should_redirect_back": {
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/private.project.2/",
			status:       http.StatusFound,
			redirectBack: true,
		},
		"subgroup_no_project_should_redirect_to_login_and_then_return404": {
			host:         "group.auth.gitlab-example.com",
			path:         "/subgroup/nonexistent/",
			status:       http.StatusNotFound,
			redirectBack: false,
		},
	}

	runPages(t, []ListenSpec{httpsListener}, certFile)

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			rsp1, err1 := GetRedirectPage(t, httpsListener, tt.host, tt.path)
			require.NoError(t, err1)
			testhelpers.Close(t, rsp1.Body)

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
			testhelpers.Close(t, rsp2.Body)

			require.Equal(t, http.StatusFound, rsp2.StatusCode)
			pagesDomainCookie := rsp2.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp1, err := GetRedirectPageWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagesDomainCookie)
			require.NoError(t, err)
			testhelpers.Close(t, authrsp1.Body)

			// Will redirect auth callback to correct host
			authLoc, err := url.Parse(authrsp1.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tt.host, authLoc.Host)
			require.Equal(t, "/auth", authLoc.Path)

			// Request auth callback in project domain
			authrsp2, err := GetRedirectPageWithCookie(t, httpsListener, authLoc.Host, authLoc.Path+"?"+authLoc.RawQuery, cookie)
			require.NoError(t, err)
			testhelpers.Close(t, authrsp2.Body)

			// server returns the ticket, user will be redirected to the project page
			require.Equal(t, http.StatusFound, authrsp2.StatusCode)
			cookie = authrsp2.Header.Get("Set-Cookie")

			rsp3, err3 := GetRedirectPageWithCookie(t, httpsListener, tt.host, tt.path, cookie)
			require.NoError(t, err3)
			testhelpers.Close(t, rsp3.Body)

			require.Equal(t, tt.status, rsp3.StatusCode)

			// Make sure there are no cache headers
			require.Empty(t, rsp3.Header.Values("Cache-Control"))
			require.Empty(t, rsp3.Header.Values("Expires"))

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
	testAccessControl(t, RunPagesProcessWithSSLCertFile)
}

func TestAccessControlWithSSLCertDir(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithSSLCertDir)
}

// This proves the fix for https://gitlab.com/gitlab-org/gitlab-pages/-/issues/262
// Read the issue description if any changes to internal/auth/ break this test.
// Related to https://tools.ietf.org/html/rfc6749#section-10.6.
func TestHijackedCode(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{proxyListener}),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)

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
	testhelpers.Close(t, maliciousResp.Body)

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
	testhelpers.Close(t, authrsp.Body)

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
	testhelpers.Close(t, impersonatingRes.Body)

	require.Equal(t, impersonatingRes.StatusCode, http.StatusInternalServerError, "should fail to decode code")
}

func getValidCookieAndState(t *testing.T, domain string) (string, string) {
	t.Helper()

	// follow flow to get a valid cookie
	// visit https://<domain>/
	rsp, err := GetProxyRedirectPageWithCookie(t, proxyListener, domain, "/", "", true)
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)

	cookie := rsp.Header.Get("Set-Cookie")
	require.NotEmpty(t, cookie)

	redirectURL, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := redirectURL.Query().Get("state")
	require.NotEmpty(t, state)

	return cookie, state
}

func defaultAuthConfig(t *testing.T) string {
	t.Helper()

	configs := []string{
		"gitlab-server=https://public-gitlab-auth.com",
		"auth-redirect-uri=https://projects.gitlab-example.com/auth",
	}

	configFile := defaultConfigFileWith(t, configs...)

	return configFile
}
