package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariablesConfig(t *testing.T) {
	t.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))

	RunPagesProcess(t,
		withoutWait,
		withListeners([]ListenSpec{}), // explicitly disable listeners for this test
	)
	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	t.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))

	RunPagesProcess(t,
		withoutWait,
		withListeners([]ListenSpec{httpsListener}),
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
	t.Setenv("LISTEN_HTTP", fmt.Sprintf("%s,%s", net.JoinHostPort("127.0.0.1", "37001"), net.JoinHostPort("127.0.0.1", "37002")))

	RunPagesProcess(t,
		withoutWait,
		withListeners([]ListenSpec{}), // explicitly disable listeners for this test
	)

	for _, listener := range listenSpecs {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestUnixSocketListener(t *testing.T) {
	tmp := t.TempDir()
	sockPath := filepath.Join(tmp, "unix.sock")

	spec := ListenSpec{
		Type: "unix",
		Host: sockPath,
	}

	RunPagesProcess(t,
		withListeners([]ListenSpec{spec}),
	)
	require.NoError(t, spec.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, spec, "group.gitlab-example.com", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}
