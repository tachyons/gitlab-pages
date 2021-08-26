package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariablesConfig(t *testing.T) {
	envVarValue := "LISTEN_HTTP=" + net.JoinHostPort(httpListener.Host, httpListener.Port)

	RunPagesProcess(t,
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

	RunPagesProcess(t,
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

	RunPagesProcess(t,
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
