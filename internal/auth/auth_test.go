package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/mock"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func createTestAuth(t *testing.T, internalServer string, publicServer string) *Auth {
	t.Helper()

	a, err := New(&Options{
		PagesDomain:          "pages.gitlab-example.com",
		StoreSecret:          "something-very-secret",
		ClientID:             "id",
		ClientSecret:         "secret",
		RedirectURI:          "http://pages.gitlab-example.com/auth",
		InternalGitlabServer: internalServer,
		PublicGitlabServer:   publicServer,
		AuthScope:            "scope",
		AuthTimeout:          5 * time.Second,
		CookieSessionTimeout: 10 * time.Minute,
	})

	require.NoError(t, err)

	return a
}

type domainMock struct {
	projectID       uint64
	notFoundContent string
}

func (dm *domainMock) GetProjectID(r *http.Request) uint64 {
	return dm.projectID
}

func (dm *domainMock) ServeNotFoundAuthFailed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(dm.notFoundContent))
}

// Gorilla's sessions use request context to save session
// Which makes session sharable between test code and actually manipulating session
// Which leads to negative side effects: we can't test encryption, and cookie params
// like max-age and secure are not being properly set
// To avoid that we use fake request, and set only session cookie without copying context
func setSessionValues(t *testing.T, r *http.Request, auth *Auth, values map[interface{}]interface{}) {
	t.Helper()

	tmpRequest, err := http.NewRequest("GET", "http://"+r.Host, nil)
	require.NoError(t, err)

	result := httptest.NewRecorder()

	session, _ := auth.getSessionFromStore(tmpRequest)
	session.Values = values
	err = session.Save(tmpRequest, result)
	require.NoError(t, err)

	res := result.Result()
	testhelpers.Close(t, res.Body)

	for _, cookie := range res.Cookies() {
		r.AddCookie(cookie)
	}
}

func TestTryAuthenticate(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something/else")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.False(t, auth.TryAuthenticate(result, r, mockSource))
}

func TestTryAuthenticateWithError(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?error=access_denied")
	require.NoError(t, err)

	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))
	require.Equal(t, http.StatusUnauthorized, result.Code)
}

func TestTryAuthenticateWithCodeButInvalidState(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/auth?code=1&state=invalid", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"state": "state",
	})

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))
	require.Equal(t, http.StatusUnauthorized, result.Code)
}

func TestTryAuthenticateRemoveTokenFromRedirect(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/auth?code=1&state=state&token=secret", nil)
	require.Equal(t, r.URL.Query().Get("token"), "secret", "token is present before redirecting")
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"state":             "state",
		"proxy_auth_domain": "https://domain.com",
	})

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))
	require.Equal(t, http.StatusFound, result.Code)

	redirect, err := url.Parse(result.Header().Get("Location"))
	require.NoError(t, err)

	require.Empty(t, redirect.Query().Get("token"), "token is gone after redirecting")
}

func TestTryAuthenticateWithDomainAndState(t *testing.T) {
	auth := createTestAuth(t, "", "public-gitlab.example.com")
	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?domain=https%3A%2F%2Fpages.gitlab-example.com&state=state")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))
	require.Equal(t, http.StatusFound, result.Code)
	redirect, err := url.Parse(result.Header().Get("Location"))
	require.NoError(t, err)

	require.Equal(t, "/public-gitlab.example.com/oauth/authorize?client_id=id&redirect_uri=http://pages.gitlab-example.com/auth&response_type=code&state=state&scope=scope", redirect.String())
}

func TestCheckAuthenticationWhenStateIsAlreadySet(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/", nil)
	require.NoError(t, err)

	// pre-set an state
	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"state": "given_state",
	})

	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)

	// check if the state was re-used instead of re-created
	session, _ := auth.getSessionFromStore(r)
	require.Equal(t, "given_state", session.Values["state"], "did not reuse the pre-set state")
}

func TestTryAuthenticateWithNonHttpDomainAndState(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/auth?domain=mailto://example.com?body=TESTBODY&state=state", nil)
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))
	require.Equal(t, http.StatusUnauthorized, result.Code)
}

func testTryAuthenticateWithCodeAndState(t *testing.T, https bool) {
	t.Helper()

	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			require.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	host := "http://example.com"
	if https {
		host = "https://example.com"
	}

	code, err := auth.EncryptAndSignCode(host, "1")
	require.NoError(t, err)

	r, err := http.NewRequest("GET", host+"/auth?code="+code+"&state=state", nil)
	require.NoError(t, err)
	if https {
		r.URL.Scheme = request.SchemeHTTPS
	} else {
		r.URL.Scheme = request.SchemeHTTP
	}

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"uri":   "https://pages.gitlab-example.com/project/",
		"state": "state",
	})

	result := httptest.NewRecorder()

	mockCtrl := gomock.NewController(t)

	mockSource := mock.NewMockSource(mockCtrl)
	require.True(t, auth.TryAuthenticate(result, r, mockSource))

	res := result.Result()
	testhelpers.Close(t, res.Body)

	require.Equal(t, http.StatusFound, result.Code)
	require.Equal(t, "https://pages.gitlab-example.com/project/", result.Header().Get("Location"))
	require.Equal(t, 600, res.Cookies()[0].MaxAge)
	require.Equal(t, https, res.Cookies()[0].Secure)
}

func TestTryAuthenticateWithCodeAndStateOverHTTP(t *testing.T) {
	testTryAuthenticateWithCodeAndState(t, false)
}

func TestTryAuthenticateWithCodeAndStateOverHTTPS(t *testing.T) {
	testTryAuthenticateWithCodeAndState(t, true)
}

func TestCheckAuthenticationWhenAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()
	r, err := http.NewRequest("Get", "https://example.com/", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{"access_token": "abc"})
	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.False(t, contentServed)

	// notFoundContent wasn't served so the default response from CheckAuthentication should be 200
	require.Equal(t, http.StatusOK, result.Code)
}

func TestCheckAuthenticationWhenContextCanceled(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(apiServer.Close)

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()
	r, err := http.NewRequest("Get", "https://example.com/", nil)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)
	setSessionValues(t, r, auth, map[interface{}]interface{}{"access_token": "abc"})

	// cancel context explicitly and expect not found
	cancel()
	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)
	require.Equal(t, http.StatusNotFound, result.Code)
}

func TestCheckAuthenticationWhenNoAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	w := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/auth?code=1&state=state", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"access_token": "abc",
	})

	contentServed := auth.CheckAuthentication(w, r, &domainMock{projectID: 1000, notFoundContent: "Generic 404"})
	require.True(t, contentServed)
	res := w.Result()
	testhelpers.Close(t, res.Body)

	require.Equal(t, http.StatusNotFound, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, string(body), "Generic 404")
}

func TestCheckAuthenticationWithSessionFromDifferentHost(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()
	r, err := http.NewRequest("Get", "https://different.com/", nil)
	require.NoError(t, err)
	setSessionValues(t, r, auth, map[interface{}]interface{}{"access_token": "abc"})

	r, err = http.NewRequest("Get", "https://example.com/", nil)
	require.NoError(t, err)
	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)

	// should redirect to auth
	require.Equal(t, http.StatusFound, result.Code)
	redirectURL, err := url.Parse(result.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "pages.gitlab-example.com", redirectURL.Host)
	require.Equal(t, "/auth", redirectURL.Path)
	require.Equal(t, "https://example.com", redirectURL.Query().Get("domain"))
}

func TestCheckAuthenticationWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"access_token": "abc",
	})

	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)
	require.Equal(t, http.StatusFound, result.Code)
}

func TestCheckAuthenticationWithoutProject(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/auth?code=1&state=state", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"access_token": "abc",
	})

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.False(t, contentServed)
	require.Equal(t, http.StatusOK, result.Code)
}

func TestCheckAuthenticationWithoutProjectWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL, "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com/", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"access_token": "abc",
	})

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.True(t, contentServed)
	require.Equal(t, http.StatusFound, result.Code)
}

func TestGenerateKeys(t *testing.T) {
	keys, err := generateKeys("something-very-secret", 3)
	require.NoError(t, err)
	require.Len(t, keys, 3)

	require.NotEqual(t, fmt.Sprint(keys[0]), fmt.Sprint(keys[1]))
	require.NotEqual(t, fmt.Sprint(keys[0]), fmt.Sprint(keys[2]))
	require.NotEqual(t, fmt.Sprint(keys[1]), fmt.Sprint(keys[2]))

	require.Len(t, keys[0], 32)
	require.Len(t, keys[1], 32)
	require.Len(t, keys[2], 32)
}

func TestGetTokenIfExistsWhenTokenExists(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{
		"access_token": "abc",
	})

	token, err := auth.GetTokenIfExists(result, r)
	require.NoError(t, err)
	require.Equal(t, "abc", token)
}

func TestGetTokenIfExistsWhenTokenDoesNotExist(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()

	r, err := http.NewRequest("Get", "https://example.com", nil)
	require.NoError(t, err)

	setSessionValues(t, r, auth, map[interface{}]interface{}{})

	token, err := auth.GetTokenIfExists(result, r)
	require.Equal(t, "", token)
	require.NoError(t, err)
}

func TestCheckResponseForInvalidTokenWhenInvalidToken(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("http://pages.gitlab-example.com/test")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL, Host: "pages.gitlab-example.com", RequestURI: "/test"}

	resp := &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(bytes.NewReader([]byte("{\"error\":\"invalid_token\"}")))}
	testhelpers.Close(t, resp.Body)

	require.True(t, auth.CheckResponseForInvalidToken(result, r, resp))
	res := result.Result()
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, "http://pages.gitlab-example.com/test", result.Header().Get("Location"))
}

func TestCheckResponseForInvalidTokenWhenNotInvalidToken(t *testing.T) {
	auth := createTestAuth(t, "", "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte("ok")))}

	require.False(t, auth.CheckResponseForInvalidToken(result, r, resp))
}

func TestDomainAllowed(t *testing.T) {
	auth := createTestAuth(t, "", "")
	mockCtrl := gomock.NewController(t)
	mockSource := mock.NewMockSource(mockCtrl)

	testCases := []struct {
		name     string
		expected bool
	}{
		{
			name:     "pages.unrelated-site.com",
			expected: false,
		},
		{
			name:     "prepended-pages.gitlab-example.com",
			expected: false,
		},
		{
			name:     "pages.gitlab-example.com.unrelated-site.com",
			expected: false,
		},
		{
			name:     "pages.gitlab-example.com",
			expected: true,
		},
		{
			name:     "subdomain.pages.gitlab-example.com",
			expected: true,
		},
		{
			name:     "multi.sub.domain.pages.gitlab-example.com",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			if !tc.expected {
				mockSource.EXPECT().GetDomain(ctx, tc.name).Return(nil, nil)
			}

			actual := auth.domainAllowed(ctx, tc.name, mockSource)
			require.Equal(t, tc.expected, actual)
		})
	}
}
