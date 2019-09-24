package auth

import (
	"fmt"
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
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	require.Equal(t, false, auth.TryAuthenticate(result, r, new(source.Domains)))
}

func TestTryAuthenticateWithError(t *testing.T) {
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?error=access_denied")
	require.NoError(t, err)
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	require.Equal(t, true, auth.TryAuthenticate(result, r, new(source.Domains)))
	require.Equal(t, 401, result.Code)
}

func TestTryAuthenticateWithCodeButInvalidState(t *testing.T) {
	store := sessions.NewCookieStore([]byte("something-very-secret"))
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=invalid")
	require.NoError(t, err)
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["state"] = "state"
	session.Save(r, result)

	require.Equal(t, true, auth.TryAuthenticate(result, r, new(source.Domains)))
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

	r, _ := http.NewRequest("GET", "/auth?code=1&state=state", nil)
	r = request.WithHTTPSFlag(r, https)

	setSessionValues(r, map[interface{}]interface{}{
		"uri":   "https://pages.gitlab-example.com/project/",
		"state": "state",
	})

	result := httptest.NewRecorder()
	require.Equal(t, true, auth.TryAuthenticate(result, r, new(source.Domains)))
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
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	require.Equal(t, false, auth.CheckAuthentication(result, r, 1000))
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
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	require.Equal(t, true, auth.CheckAuthentication(result, r, 1000))
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
	r = request.WithHTTPSFlag(r, false)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	require.Equal(t, true, auth.CheckAuthentication(result, r, 1000))
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
	r := request.WithHTTPSFlag(&http.Request{URL: reqURL}, true)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	require.Equal(t, false, auth.CheckAuthenticationWithoutProject(result, r))
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
	r = request.WithHTTPSFlag(r, false)

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	require.Equal(t, true, auth.CheckAuthenticationWithoutProject(result, r))
	require.Equal(t, 302, result.Code)
}

func TestGenerateKeyPair(t *testing.T) {
	signingSecret, encryptionSecret := generateKeyPair("something-very-secret")
	require.NotEqual(t, fmt.Sprint(signingSecret), fmt.Sprint(encryptionSecret))
	require.Equal(t, len(signingSecret), 32)
	require.Equal(t, len(encryptionSecret), 32)
}
