package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariablesConfig(t *testing.T) {
	skipUnlessEnabled(t)
	os.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()
	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com:", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	skipUnlessEnabled(t)
	os.Setenv("LISTEN_HTTP", net.JoinHostPort(httpListener.Host, httpListener.Port))
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, []ListenSpec{httpsListener}, "")
	defer teardown()

	for _, listener := range []ListenSpec{httpListener, httpsListener} {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")
		require.NoError(t, err)
		rsp.Body.Close()

		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}

func TestMultiFlagEnvironmentVariables(t *testing.T) {
	skipUnlessEnabled(t)
	listenSpecs := []ListenSpec{{"http", "127.0.0.1", "37001"}, {"http", "127.0.0.1", "37002"}}
	envVarValue := fmt.Sprintf("%s,%s", net.JoinHostPort("127.0.0.1", "37001"), net.JoinHostPort("127.0.0.1", "37002"))

	os.Setenv("LISTEN_HTTP", envVarValue)
	defer func() { os.Unsetenv("LISTEN_HTTP") }()

	teardown := RunPagesProcess(t, *pagesBinary, []ListenSpec{}, "")
	defer teardown()

	for _, listener := range listenSpecs {
		require.NoError(t, listener.WaitUntilRequestSucceeds(nil))
		rsp, err := GetPageFromListener(t, listener, "group.gitlab-example.com", "project/")

		require.NoError(t, err)
		rsp.Body.Close()
		require.Equal(t, http.StatusOK, rsp.StatusCode)
	}
}
