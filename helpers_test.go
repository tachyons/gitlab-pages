package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
)

const apiServerListenAddress = "127.0.0.1:7800"

type tWriter struct {
	t *testing.T
}

func (t *tWriter) Write(b []byte) (int, error) {
	t.t.Log(string(bytes.TrimRight(b, "\r\n")))

	return len(b), nil
}

// The HTTPS certificate isn't signed by anyone. This http client is set up
// so it can talk to servers using it.
var (
	TestHTTPSClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: TestCertPool},
		},
	}

	// Use HTTP with a very short timeout to repeatedly check for the server to be
	// up. Again, ignore HTTP
	QuickTimeoutHTTPSClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{RootCAs: TestCertPool},
			ResponseHeaderTimeout: 100 * time.Millisecond,
		},
	}

	TestCertPool = x509.NewCertPool()
)

func init() {
	if ok := TestCertPool.AppendCertsFromPEM([]byte(fixture.Certificate)); !ok {
		fmt.Println("Failed to load cert!")
	}
}

func CreateHTTPSFixtureFiles(t *testing.T) (key string, cert string) {
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

// ListenSpec is used to point at a gitlab-pages http server, preserving the
// type of port it is (http, https, proxy)
type ListenSpec struct {
	Type string
	Host string
	Port string
}

func (l ListenSpec) URL(suffix string) string {
	scheme := "http"
	if l.Type == "https" {
		scheme = "https"
	}

	suffix = strings.TrimPrefix(suffix, "/")

	return fmt.Sprintf("%s://%s/%s", scheme, l.JoinHostPort(), suffix)
}

// Returns only once this spec points at a working TCP server
func (l ListenSpec) WaitUntilRequestSucceeds(done chan struct{}) error {
	timeout := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		select {
		case <-done:
			return fmt.Errorf("server has shut down already")
		default:
		}

		req, err := http.NewRequest("GET", l.URL("/"), nil)
		if err != nil {
			return err
		}

		response, err := QuickTimeoutHTTPSClient.Transport.RoundTrip(req)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		response.Body.Close()

		if code := response.StatusCode; code >= 200 && code < 500 {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timed out after %v waiting for listener %v", timeout, l)
}

func (l ListenSpec) JoinHostPort() string {
	return net.JoinHostPort(l.Host, l.Port)
}

// RunPagesProcess will start a gitlab-pages process with the specified listeners
// and return a function you can call to shut it down again. Use
// GetPageFromProcess to do a HTTP GET against a listener.
//
// If run as root via sudo, the gitlab-pages process will drop privileges
func RunPagesProcess(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, extraArgs ...string) (teardown func()) {
	return runPagesProcess(t, true, pagesPath, listeners, promPort, nil, extraArgs...)
}

func RunPagesProcessWithoutWait(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, extraArgs ...string) (teardown func()) {
	return runPagesProcess(t, false, pagesPath, listeners, promPort, nil, extraArgs...)
}

func RunPagesProcessWithSSLCertFile(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, extraArgs ...string) (teardown func()) {
	return runPagesProcess(t, true, pagesPath, listeners, promPort, []string{"SSL_CERT_FILE=" + sslCertFile}, extraArgs...)
}

func RunPagesProcessWithAuth(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string) (teardown func()) {
	return runPagesProcess(t, true, pagesPath, listeners, promPort, nil, "-auth-client-id=1",
		"-auth-client-secret=1",
		"-auth-server=https://gitlab-auth.com",
		"-auth-redirect-uri=https://projects.gitlab-example.com/auth",
		"-auth-secret=something-very-secret")
}

func RunPagesProcessWithAuthServer(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, authServer string) (teardown func()) {
	return runPagesProcess(t, true, pagesPath, listeners, promPort, nil, "-auth-client-id=1",
		"-auth-client-secret=1",
		"-auth-server="+authServer,
		"-auth-redirect-uri=https://projects.gitlab-example.com/auth",
		"-auth-secret=something-very-secret")
}

func RunPagesProcessWithAuthServerWithSSL(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, authServer string) (teardown func()) {
	return runPagesProcess(t, true, pagesPath, listeners, promPort, []string{"SSL_CERT_FILE=" + sslCertFile}, "-auth-client-id=1",
		"-auth-client-secret=1",
		"-auth-server="+authServer,
		"-auth-redirect-uri=https://projects.gitlab-example.com/auth",
		"-auth-secret=something-very-secret")
}

func runPagesProcess(t *testing.T, wait bool, pagesPath string, listeners []ListenSpec, promPort string, extraEnv []string, extraArgs ...string) (teardown func()) {
	_, err := os.Stat(pagesPath)
	require.NoError(t, err)

	apiServer := &http.Server{
		Addr:    apiServerListenAddress,
		Handler: http.HandlerFunc(fixture.MockHTTPHandler),
	}

	args, tempfiles := getPagesArgs(t, listeners, promPort, extraArgs)
	args = append(args, "-api-server", "http://"+apiServerListenAddress+"/api/v4")
	cmd := exec.Command(pagesPath, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = &tWriter{t}
	cmd.Stderr = &tWriter{t}
	require.NoError(t, cmd.Start())
	t.Logf("Running %s %v", pagesPath, args)

	waitCh := make(chan struct{})
	go func() {
		apiServer.ListenAndServe()
		cmd.Wait()
		for _, tempfile := range tempfiles {
			os.Remove(tempfile)
		}
		close(waitCh)
	}()

	cleanup := func() {
		apiServer.Close()
		cmd.Process.Signal(os.Interrupt)
		<-waitCh
	}

	if wait {
		for _, spec := range listeners {
			if err := spec.WaitUntilRequestSucceeds(waitCh); err != nil {
				cleanup()
				t.Fatal(err)
			}
		}
	}

	return cleanup
}

func getPagesArgs(t *testing.T, listeners []ListenSpec, promPort string, extraArgs []string) (args, tempfiles []string) {
	var hasHTTPS bool

	args = append(args, "-log-verbose=true")

	for _, spec := range listeners {
		args = append(args, "-listen-"+spec.Type, spec.JoinHostPort())

		if spec.Type == "https" {
			hasHTTPS = true
		}
	}

	if hasHTTPS {
		key, cert := CreateHTTPSFixtureFiles(t)
		tempfiles = []string{key, cert}
		args = append(args, "-root-key", key, "-root-cert", cert)
	}

	if promPort != "" {
		args = append(args, "-metrics-address", promPort)
	}

	args = append(args, getPagesDaemonArgs(t)...)
	args = append(args, extraArgs...)

	return
}

func getPagesDaemonArgs(t *testing.T) []string {
	mode := os.Getenv("TEST_DAEMONIZE")
	if mode == "" {
		return nil
	}

	if os.Geteuid() != 0 {
		t.Log("Privilege-dropping requested but not running as root!")
		t.FailNow()
		return nil
	}

	out := []string{}

	switch mode {
	case "tmpdir":
		out = append(out, "-daemon-inplace-chroot=false")
	case "inplace":
		out = append(out, "-daemon-inplace-chroot=true")
	default:
		t.Log("Unknown daemonize mode", mode)
		t.FailNow()
		return nil
	}

	t.Log("Running pages as a daemon")

	// This triggers the drop-privileges-and-chroot code in the pages daemon
	out = append(out, "-daemon-uid", "0")
	out = append(out, "-daemon-gid", "65534")

	return out
}

// Does a HTTP(S) GET against the listener specified, setting a fake
// Host: and constructing the URL from the listener and the URL suffix.
func GetPageFromListener(t *testing.T, spec ListenSpec, host, urlsuffix string) (*http.Response, error) {
	return GetPageFromListenerWithCookie(t, spec, host, urlsuffix, "")
}

func GetPageFromListenerWithCookie(t *testing.T, spec ListenSpec, host, urlsuffix string, cookie string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	req.Host = host

	return DoPagesRequest(t, req)
}

func GetProxiedPageFromListener(t *testing.T, spec ListenSpec, host, xForwardedHost, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host
	req.Header.Set("X-Forwarded-Host", xForwardedHost)

	return DoPagesRequest(t, req)
}

func DoPagesRequest(t *testing.T, req *http.Request) (*http.Response, error) {
	t.Logf("curl -X %s -H'Host: %s' %s", req.Method, req.Host, req.URL)

	return TestHTTPSClient.Do(req)
}

func GetRedirectPage(t *testing.T, spec ListenSpec, host, urlsuffix string) (*http.Response, error) {
	return GetRedirectPageWithCookie(t, spec, host, urlsuffix, "")
}

func GetRedirectPageWithCookie(t *testing.T, spec ListenSpec, host, urlsuffix string, cookie string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	req.Host = host

	return TestHTTPSClient.Transport.RoundTrip(req)
}

func waitForRoundtrips(t *testing.T, listeners []ListenSpec, timeout time.Duration) {
	nListening := 0
	start := time.Now()
	for _, spec := range listeners {
		for time.Since(start) < timeout {
			req, err := http.NewRequest("GET", spec.URL("/"), nil)
			if err != nil {
				t.Fatal(err)
			}

			if response, err := QuickTimeoutHTTPSClient.Transport.RoundTrip(req); err == nil {
				nListening++
				response.Body.Close()
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	require.Equal(t, len(listeners), nListening, "all listeners must be accepting TCP connections")
}
