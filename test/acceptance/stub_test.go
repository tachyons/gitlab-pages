package acceptance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
)

var defaultProcessConfig = processConfig{
	wait:           true,
	pagesBinary:    *pagesBinary,
	listeners:      supportedListeners(),
	envs:           []string{},
	extraArgs:      []string{},
	gitlabStubOpts: &stubOpts{},
}

type processConfig struct {
	wait           bool
	pagesBinary    string
	listeners      []ListenSpec
	envs           []string
	extraArgs      []string
	gitlabStubOpts *stubOpts
}

type processOption func(*processConfig)

func withoutWait(config *processConfig) {
	config.wait = false
}

func withListeners(listeners []ListenSpec) processOption {
	return func(config *processConfig) {
		config.listeners = listeners
	}
}

func withEnv(envs []string) processOption {
	return func(config *processConfig) {
		config.envs = append(config.envs, envs...)
	}
}

func withExtraArgument(key, value string) processOption {
	return func(config *processConfig) {
		config.extraArgs = append(config.extraArgs, fmt.Sprintf("-%s=%s", key, value))
	}
}
func withArguments(args []string) processOption {
	return func(config *processConfig) {
		config.extraArgs = append(config.extraArgs, args...)
	}
}

func withStubOptions(opts *stubOpts) processOption {
	return func(config *processConfig) {
		config.gitlabStubOpts = opts
	}
}

// makeGitLabPagesAccessStub provides a stub *httptest.Server to check pages_access API call.
// the result is based on the project id.
//
// Project IDs must be 4 digit long and the following rules applies:
//   1000-1999: Ok
//   2000-2999: Unauthorized
//   3000-3999: Invalid token
func makeGitLabPagesAccessStub(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewUnstartedServer(apiHandler(t))
}

func apiHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// TODO: move OAuth and user endpoints to NewGitlabDomainsSourceStub
		case "/oauth/token":
			require.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		case "/api/v4/internal/pages/status":
			// Temporarily adding these handlers to this stub.
			w.WriteHeader(http.StatusNoContent)
		case "/api/v4/internal/pages":
			defaultAPIHandler(t, &stubOpts{})(w, r)
		default:
			if handleAccessControlArtifactRequests(t, w, r) {
				return
			}
			handleAccessControlRequests(t, w, r)
		}
	}
}

func CreateHTTPSFixtureFiles(t *testing.T) (key string, cert string) {
	t.Helper()

	tmpDir := t.TempDir()

	keyfile, err := os.CreateTemp(tmpDir, "https-fixture")
	require.NoError(t, err)
	key = keyfile.Name()
	keyfile.Close()

	certfile, err := os.CreateTemp(tmpDir, "https-fixture")
	require.NoError(t, err)
	cert = certfile.Name()
	certfile.Close()

	require.NoError(t, os.WriteFile(key, []byte(fixture.Key), 0644))
	require.NoError(t, os.WriteFile(cert, []byte(fixture.Certificate), 0644))

	return keyfile.Name(), certfile.Name()
}

func CreateGitLabAPISecretKeyFixtureFile(t *testing.T) (filepath string) {
	t.Helper()

	secretfile, err := os.CreateTemp("", "gitlab-api-secret")
	require.NoError(t, err)
	secretfile.Close()

	require.NoError(t, os.WriteFile(secretfile.Name(), []byte(fixture.GitLabAPISecretKey), 0644))

	return secretfile.Name()
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
