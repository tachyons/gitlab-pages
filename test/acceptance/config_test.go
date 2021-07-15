package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariablesConfig(t *testing.T) {
	envVarValue := "LISTEN_HTTP=" + net.JoinHostPort(httpListener.Host, httpListener.Port)

	RunPagesProcessWithStubGitLabServer(t,
		withoutWait,
		withListeners([]ListenSpec{}), // explicitly disable listeners for this test
		withEnv([]string{envVarValue}),
	)
	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	envVarValue := "LISTEN_HTTP=" + net.JoinHostPort(httpListener.Host, httpListener.Port)

	RunPagesProcessWithStubGitLabServer(t,
		withoutWait,
		withListeners([]ListenSpec{httpsListener}),
		withEnv([]string{envVarValue}),
	)

	for _, listener := range []ListenSpec{httpListener, httpsListener} {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		rsp.Body.Close()

		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestMultipleListenersFromEnvironmentVariables(t *testing.T) {
	listenSpecs := []ListenSpec{{"http", "127.0.0.1", "37001"}, {"http", "127.0.0.1", "37002"}}
	envVarValue := fmt.Sprintf("LISTEN_HTTP=%s,%s", net.JoinHostPort("127.0.0.1", "37001"), net.JoinHostPort("127.0.0.1", "37002"))

	RunPagesProcessWithStubGitLabServer(t,
		withoutWait,
		withListeners([]ListenSpec{}), // explicitly disable listeners for this test
		withEnv([]string{envVarValue}),
	)

	for _, listener := range listenSpecs {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

// TODO: remove along chroot https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
func TestEnableJailFromEnvironment(t *testing.T) {
	out, teardown := runPagesProcess(t,
		true,
		*pagesBinary,
		[]ListenSpec{httpListener},
		"",
		[]string{
			"DAEMON_ENABLE_JAIL=true",
		},
		"-domain-config-source", "disk",
	)
	t.Cleanup(teardown)

	require.Eventually(t, func() bool {
		require.Contains(t, out.String(), "\"daemon-enable-jail\":true")
		return true
	}, time.Second, 10*time.Millisecond)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}
