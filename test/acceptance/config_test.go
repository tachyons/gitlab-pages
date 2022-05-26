package acceptance_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
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

func TestMetricsHTTPSConfig(t *testing.T) {
	keyFile, certFile := CreateHTTPSFixtureFiles(t)

	RunPagesProcess(t,
		withExtraArgument("metrics-address", ":42345"),
		withExtraArgument("metrics-certificate", certFile),
		withExtraArgument("metrics-key", keyFile),
	)
	require.NoError(t, httpsListener.WaitUntilRequestSucceeds(nil))

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	res, err := client.Get("https://127.0.0.1:42345/metrics")
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)

	res, err = client.Get("http://127.0.0.1:42345/metrics")
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}
