package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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

func withListeners(listeners []ListenSpec) processOption {
	return func(config *processConfig) {
		config.listeners = listeners
	}
}

func withEnv(envs []string) processOption {
	return func(config *processConfig) {
		config.envs = envs
	}
}

func withExtraArgument(key, value string) processOption {
	return func(config *processConfig) {
		config.extraArgs = append(config.extraArgs, fmt.Sprintf("-%s=%s", key, value))
	}
}
func withArguments(args []string) processOption {
	return func(config *processConfig) {
		config.extraArgs = args
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

	return httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			require.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			if handleAccessControlArtifactRequests(t, w, r) {
				return
			}
			handleAccessControlRequests(t, w, r)
		}
	}))
}

func CreateHTTPSFixtureFiles(t *testing.T) (key string, cert string) {
	t.Helper()

	keyfile, err := ioutil.TempFile("", "https-fixture")
	require.NoError(t, err)
	key = keyfile.Name()
	keyfile.Close()

	certfile, err := ioutil.TempFile("", "https-fixture")
	require.NoError(t, err)
	cert = certfile.Name()
	certfile.Close()

	require.NoError(t, ioutil.WriteFile(key, []byte(fixture.Key), 0644))
	require.NoError(t, ioutil.WriteFile(cert, []byte(fixture.Certificate), 0644))

	return keyfile.Name(), certfile.Name()
}

func CreateGitLabAPISecretKeyFixtureFile(t *testing.T) (filepath string) {
	t.Helper()

	secretfile, err := ioutil.TempFile("", "gitlab-api-secret")
	require.NoError(t, err)
	secretfile.Close()

	require.NoError(t, ioutil.WriteFile(secretfile.Name(), []byte(fixture.GitLabAPISecretKey), 0644))

	return secretfile.Name()
}
