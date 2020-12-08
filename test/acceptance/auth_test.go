package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWhenAuthIsDisabledPrivateIsNotAccessible(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)
}

func TestWhenAuthIsEnabledPrivateWillRedirectToAuthorize(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

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
	require.Equal(t, "gitlab-auth.com", url.Host)
	require.Equal(t, "/oauth/authorize", url.Path)
	require.Equal(t, "clientID", url.Query().Get("client_id"))
	require.Equal(t, "https://projects.gitlab-example.com/auth", url.Query().Get("redirect_uri"))
	require.NotEqual(t, "", url.Query().Get("state"))
}

func TestWhenAuthDeniedWillCauseUnauthorized(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?error=access_denied")

	require.NoError(t, err)
	defer rsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
}
func TestWhenLoginCallbackWithWrongStateShouldFail(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

	rsp, err := GetRedirectPage(t, httpsListener, "group.auth.gitlab-example.com", "private.project/")

	require.NoError(t, err)
	defer rsp.Body.Close()

	// Go to auth page with wrong state will cause failure
	authrsp, err := GetPageFromListener(t, httpsListener, "projects.gitlab-example.com", "/auth?code=0&state=0")

	require.NoError(t, err)
	defer authrsp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, authrsp.StatusCode)
}

func TestWhenLoginCallbackWithCorrectStateWithoutEndpoint(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
	defer teardown()

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

	// Will cause 503 because token endpoint is not available
	require.Equal(t, http.StatusServiceUnavailable, authrsp.StatusCode)
}

func handleAccessControlArtifactRequests(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	authorization := r.Header.Get("Authorization")

	switch {
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/delayed_200.html`).MatchString(r.URL.Path):
		sleepIfAuthorized(t, authorization, w)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/404.html`).MatchString(r.URL.Path):
		w.WriteHeader(http.StatusNotFound)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/500.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusInternalServerError)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusOK)
		return true
	case regexp.MustCompile(`/api/v4/projects/group/subgroup/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(t, authorization, w, http.StatusOK)
		return true
	default:
		return false
	}
}

func handleAccessControlRequests(t *testing.T, w http.ResponseWriter, r *http.Request) {
	allowedProjects := regexp.MustCompile(`/api/v4/projects/1\d{3}/pages_access`)
	deniedProjects := regexp.MustCompile(`/api/v4/projects/2\d{3}/pages_access`)
	invalidTokenProjects := regexp.MustCompile(`/api/v4/projects/3\d{3}/pages_access`)

	switch {
	case allowedProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	case deniedProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
	case invalidTokenProjects.MatchString(r.URL.Path):
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
	default:
		t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
	}
}

func returnIfAuthorized(t *testing.T, authorization string, w http.ResponseWriter, status int) {
	if authorization != "" {
		require.Equal(t, "Bearer abc", authorization)
		w.WriteHeader(status)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func sleepIfAuthorized(t *testing.T, authorization string, w http.ResponseWriter) {
	if authorization != "" {
		require.Equal(t, "Bearer abc", authorization)
		time.Sleep(2 * time.Second)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestAccessControlUnderCustomDomain(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")

	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

	rsp, err := GetRedirectPage(t, httpListener, "private.domain.com", "/")
	require.NoError(t, err)
	defer rsp.Body.Close()

	cookie := rsp.Header.Get("Set-Cookie")

	url, err := url.Parse(rsp.Header.Get("Location"))
	require.NoError(t, err)

	state := url.Query().Get("state")
	require.Equal(t, url.Query().Get("domain"), "http://private.domain.com")

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
	require.Equal(t, "private.domain.com", url.Host)
	require.Equal(t, "1", url.Query().Get("code"))
	require.Equal(t, state, url.Query().Get("state"))

	// Run auth callback in custom domain
	authrsp, err = GetRedirectPageWithCookie(t, httpListener, "private.domain.com", "/auth?code=1&state="+
		state, cookie)

	require.NoError(t, err)
	defer authrsp.Body.Close()

	// Will redirect to the page
	cookie = authrsp.Header.Get("Set-Cookie")
	require.Equal(t, http.StatusFound, authrsp.StatusCode)

	url, err = url.Parse(authrsp.Header.Get("Location"))
	require.NoError(t, err)

	// Will redirect to custom domain
	require.Equal(t, "http://private.domain.com/", url.String())

	// Fetch page in custom domain
	authrsp, err = GetRedirectPageWithCookie(t, httpListener, "private.domain.com", "/", cookie)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, authrsp.StatusCode)
}

func TestCustomErrorPageWithAuth(t *testing.T) {
	skipUnlessEnabled(t, "not-inplace-chroot")
	testServer := makeGitLabPagesAccessStub(t)
	testServer.Start()
	defer testServer.Close()

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
	defer teardown()

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
			// project ID is 2000
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
			require.Equal(t, "1", url.Query().Get("code"))
			require.Equal(t, state, url.Query().Get("state"))

			// Run auth callback in custom domain
			authrsp, err = GetRedirectPageWithCookie(t, httpListener, tt.domain, "/auth?code=1&state="+
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

	teardown := RunPagesProcessWithAuthServer(t, *pagesBinary, listeners, "", testServer.URL)
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
	require.Equal(t, "1", url.Query().Get("code"))
	require.Equal(t, state, url.Query().Get("state"))

	// Run auth callback in custom domain
	authrsp, err = GetProxyRedirectPageWithCookie(t, proxyListener, "private.domain.com",
		"/auth?code=1&state="+state, cookie, true)

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
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
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
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithAuth(t, *pagesBinary, listeners, "")
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
	defer os.Remove(keyFile)
	defer os.Remove(certFile)

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
			teardown := runPages(t, *pagesBinary, listeners, "", certFile, testServer.URL)
			defer teardown()

			rsp, err := GetRedirectPage(t, httpsListener, tt.host, tt.path)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusFound, rsp.StatusCode)
			cookie := rsp.Header.Get("Set-Cookie")

			// Redirects to the projects under gitlab pages domain for authentication flow
			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "projects.gitlab-example.com", url.Host)
			require.Equal(t, "/auth", url.Path)
			state := url.Query().Get("state")

			rsp, err = GetRedirectPage(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusFound, rsp.StatusCode)
			pagesDomainCookie := rsp.Header.Get("Set-Cookie")

			// Go to auth page with correct state will cause fetching the token
			authrsp, err := GetRedirectPageWithCookie(t, httpsListener, "projects.gitlab-example.com", "/auth?code=1&state="+
				state, pagesDomainCookie)

			require.NoError(t, err)
			defer authrsp.Body.Close()

			// Will redirect auth callback to correct host
			url, err = url.Parse(authrsp.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tt.host, url.Host)
			require.Equal(t, "/auth", url.Path)

			// Request auth callback in project domain
			authrsp, err = GetRedirectPageWithCookie(t, httpsListener, url.Host, url.Path+"?"+url.RawQuery, cookie)
			require.NoError(t, err)

			// server returns the ticket, user will be redirected to the project page
			require.Equal(t, http.StatusFound, authrsp.StatusCode)
			cookie = authrsp.Header.Get("Set-Cookie")
			rsp, err = GetRedirectPageWithCookie(t, httpsListener, tt.host, tt.path, cookie)

			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, tt.status, rsp.StatusCode)
			require.Equal(t, "", rsp.Header.Get("Cache-Control"))

			if tt.redirectBack {
				url, err = url.Parse(rsp.Header.Get("Location"))
				require.NoError(t, err)

				require.Equal(t, "https", url.Scheme)
				require.Equal(t, tt.host, url.Host)
				require.Equal(t, tt.path, url.Path)
			}
		})
	}
}

func TestAccessControlWithSSLCertFile(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithAuthServerWithSSLCertFile)
}

func TestAccessControlWithSSLCertDir(t *testing.T) {
	testAccessControl(t, RunPagesProcessWithAuthServerWithSSLCertDir)
}