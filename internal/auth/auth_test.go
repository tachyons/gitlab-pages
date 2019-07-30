package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
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

func TestTryAuthenticate(t *testing.T) {
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something/else")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	assert.Equal(t, false, auth.TryAuthenticate(result, r, make(domain.Map), &sync.RWMutex{}))
}

func TestTryAuthenticateWithError(t *testing.T) {
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?error=access_denied")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	assert.Equal(t, true, auth.TryAuthenticate(result, r, make(domain.Map), &sync.RWMutex{}))
	assert.Equal(t, 401, result.Code)
}

func TestTryAuthenticateWithCodeButInvalidState(t *testing.T) {
	store := sessions.NewCookieStore([]byte("something-very-secret"))
	auth := createAuth(t)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=invalid")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["state"] = "state"
	session.Save(r, result)

	assert.Equal(t, true, auth.TryAuthenticate(result, r, make(domain.Map), &sync.RWMutex{}))
	assert.Equal(t, 401, result.Code)
}

func TestTryAuthenticateWithCodeAndState(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/projects/1000/pages_access":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["uri"] = "http://pages.gitlab-example.com/project/"
	session.Values["state"] = "state"
	session.Save(r, result)

	assert.Equal(t, true, auth.TryAuthenticate(result, r, make(domain.Map), &sync.RWMutex{}))
	assert.Equal(t, 302, result.Code)
	assert.Equal(t, "http://pages.gitlab-example.com/project/", result.Header().Get("Location"))
}

func TestCheckAuthenticationWhenAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	assert.Equal(t, false, auth.CheckAuthentication(result, r, 1000))
	assert.Equal(t, 200, result.Code)
}

func TestCheckAuthenticationWhenNoAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	assert.Equal(t, true, auth.CheckAuthentication(result, r, 1000))
	assert.Equal(t, 404, result.Code)
}

func TestCheckAuthenticationWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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

	assert.Equal(t, true, auth.CheckAuthentication(result, r, 1000))
	assert.Equal(t, 302, result.Code)
}

func TestCheckAuthenticationWithoutProject(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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
	r := &http.Request{URL: reqURL}

	session, _ := store.Get(r, "gitlab-pages")
	session.Values["access_token"] = "abc"
	session.Save(r, result)

	assert.Equal(t, false, auth.CheckAuthenticationWithoutProject(result, r))
	assert.Equal(t, 200, result.Code)
}

func TestCheckAuthenticationWithoutProjectWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			assert.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
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

	assert.Equal(t, true, auth.CheckAuthenticationWithoutProject(result, r))
	assert.Equal(t, 302, result.Code)
}

func TestGenerateKeyPair(t *testing.T) {
	signingSecret, encryptionSecret := generateKeyPair("something-very-secret")
	assert.NotEqual(t, fmt.Sprint(signingSecret), fmt.Sprint(encryptionSecret))
	assert.Equal(t, len(signingSecret), 32)
	assert.Equal(t, len(encryptionSecret), 32)
}
