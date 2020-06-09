package auth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func createAuth(t *testing.T) *Auth {
	return New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		"http://gitlab-example.com")
}

func defaultCookieStore() sessions.Store {
	return createCookieStore("something-very-secret")
}

type domainMock struct {
	projectID uint64
}

func (dm *domainMock) GetProjectID(r *http.Request) uint64 {
	return dm.projectID
}

func (dm *domainMock) ServeNotFoundAuthFailed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

// Gorilla's sessions use request context to save session
// Which makes session sharable between test code and actually manipulating session
// Which leads to negative side effects: we can't test encryption, and cookie params
// like max-age and secure are not being properly set
// To avoid that we use fake request, and set only session cookie without copying context
func setSessionValues(r *http.Request, values map[interface{}]interface{}) {
	tmpRequest, _ := http.NewRequest("GET", "/", nil)
	result := httptest.NewRecorder()
	store := defaultCookieStore()

	session, _ := store.Get(tmpRequest, "gitlab-pages")
	session.Values = values
	session.Save(tmpRequest, result)

	for _, cookie := range result.Result().Cookies() {
		r.AddCookie(cookie)
	}
}

func TestTryAuthenticate(t *testing.T) {
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something/else")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	require.Equal(t, false, auth.TryAuthenticate(result, r, source.NewMockSource()))
}

func TestTryAuthenticateWithError(t *testing.T) {
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?error=access_denied")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, 401, result.Code)
}

func TestTryAuthenticateWithCodeButInvalidState(t *testing.T) {
	store := sessions.NewCookieStore([]byte("something-very-secret"))
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=invalid")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["state"] = "state"
	session.Save(r, result)

	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, 401, result.Code)
}

func testTryAuthenticateWithCodeAndState(t *testing.T, https bool) {
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

	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	r, err := http.NewRequest("GET", "/auth?code=1&state=state", nil)
	require.NoError(t, err)
	if https {
		r.URL.Scheme = request.SchemeHTTPS
	} else {
		r.URL.Scheme = request.SchemeHTTP
	}

	setSessionValues(r, map[interface{}]interface{}{
		"uri":   "https://pages.gitlab-example.com/project/",
		"state": "state",
	})

	result := httptest.NewRecorder()
	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, 302, result.Code)
	require.Equal(t, "https://pages.gitlab-example.com/project/", result.Header().Get("Location"))
	require.Equal(t, 600, result.Result().Cookies()[0].MaxAge)
	require.Equal(t, https, result.Result().Cookies()[0].Secure)
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

	store := defaultCookieStore()
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)
	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.False(t, contentServed)

	// content wasn't served so the default response from CheckAuthentication should be 200
	require.Equal(t, 200, result.Code)
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

	store := defaultCookieStore()
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)
	// content wasn't served so the default response from CheckAuthentication should be 200
	require.Equal(t, 404, result.Code)
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

	store := defaultCookieStore()
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)
	require.Equal(t, 302, result.Code)
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

	store := defaultCookieStore()
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.False(t, contentServed)
	require.Equal(t, 200, result.Code)
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

	store := defaultCookieStore()
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}
	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.True(t, contentServed)
	require.Equal(t, 302, result.Code)
}

func TestGenerateKeyPair(t *testing.T) {
	signingSecret, encryptionSecret := generateKeyPair("something-very-secret")
	require.NotEqual(t, fmt.Sprint(signingSecret), fmt.Sprint(encryptionSecret))
	require.Equal(t, len(signingSecret), 32)
	require.Equal(t, len(encryptionSecret), 32)
}

func TestGetTokenIfExistsWhenTokenExists(t *testing.T) {
	store := sessions.NewCookieStore([]byte("something-very-secret"))
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		"")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	token, err := auth.GetTokenIfExists(result, r)
	require.NoError(t, err)
	require.Equal(t, "abc", token)
}

func TestGetTokenIfExistsWhenTokenDoesNotExist(t *testing.T) {
	store := sessions.NewCookieStore([]byte("something-very-secret"))
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		"")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("http://pages.gitlab-example.com/test")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL, Host: "pages.gitlab-example.com", RequestURI: "/test"}

	session, _ := store.Get(r, "gitlab-pages")
	session.Save(r, result)

	token, err := auth.GetTokenIfExists(result, r)
	require.Equal(t, "", token)
	require.Equal(t, nil, err)
}

func TestCheckResponseForInvalidTokenWhenInvalidToken(t *testing.T) {
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		"")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("http://pages.gitlab-example.com/test")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL, Host: "pages.gitlab-example.com", RequestURI: "/test"}

	resp := &http.Response{StatusCode: http.StatusUnauthorized, Body: ioutil.NopCloser(bytes.NewReader([]byte("{\"error\":\"invalid_token\"}")))}

	require.Equal(t, true, auth.CheckResponseForInvalidToken(result, r, resp))
	require.Equal(t, http.StatusFound, result.Result().StatusCode)
	require.Equal(t, "http://pages.gitlab-example.com/test", result.Header().Get("Location"))
}

func TestCheckResponseForInvalidTokenWhenNotInvalidToken(t *testing.T) {
	auth := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		"")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	resp := &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewReader([]byte("ok")))}

	require.Equal(t, false, auth.CheckResponseForInvalidToken(result, r, resp))
}
