package acceptance_test

import (
	"fmt"
	"net/http"
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

func CreateHTTPSFixtureFiles(t *testing.T) (key string, cert string) {
	t.Helper()

	keyfile, err := os.CreateTemp("", "https-fixture")
	require.NoError(t, err)
	key = keyfile.Name()
	keyfile.Close()

	certfile, err := os.CreateTemp("", "https-fixture")
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
