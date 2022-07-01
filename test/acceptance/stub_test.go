package acceptance_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/test/gitlabstub"
)

var defaultProcessConfig = processConfig{
	wait:           true,
	pagesBinary:    *pagesBinary,
	listeners:      supportedListeners(),
	extraArgs:      []string{},
	gitlabStubOpts: []gitlabstub.Option{},
}

type processConfig struct {
	wait           bool
	pagesBinary    string
	listeners      []ListenSpec
	extraArgs      []string
	gitlabStubOpts []gitlabstub.Option
	publicServer   bool
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

func withPublicServer(config *processConfig) {
	config.publicServer = true
}

func withStubOptions(opts ...gitlabstub.Option) processOption {
	return func(config *processConfig) {
		config.gitlabStubOpts = opts
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
