package acceptance_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariablesConfig(t *testing.T) {
	skipUnlessEnabled(t)

	envVarValue := "LISTEN_HTTP=" + net.JoinHostPort(httpListener.Host, httpListener.Port)

	RunPagesProcessWithStubGitLabServer(t,
		withoutWait,
		withListeners([]ListenSpec{}), // explicitly disable listeners for this test
		withEnv([]string{envVarValue}),
	)
	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com:", "project/")

	require.NoError(t, err)
	rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestMixedConfigSources(t *testing.T) {
	skipUnlessEnabled(t)
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
	skipUnlessEnabled(t)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()

		fmt.Println("checking netstat...")
		cmd := exec.Command("netstat", "-plnut")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Printf("NETSTAT FAILED: %+v\n", err)
		}
	}()

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
	wg.Done()

	wg.Wait()
}
