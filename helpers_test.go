package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

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

func CreateGitLabAPISecretKeyFixtureFile(t *testing.T) (filepath string) {
	secretfile, err := ioutil.TempFile("", "gitlab-api-secret")
	require.NoError(t, err)
	secretfile.Close()

	require.NoError(t, ioutil.WriteFile(secretfile.Name(), []byte(fixture.GitLabAPISecretKey), 0644))

	return secretfile.Name()
}

// ListenSpec is used to point at a gitlab-pages http server, preserving the
// type of port it is (http, https, proxy)
type ListenSpec struct {
	Type string
	Host string
	Port string
}

func (l ListenSpec) URL(suffix string) string {
	scheme := request.SchemeHTTP
	if l.Type == request.SchemeHTTPS {
		scheme = request.SchemeHTTPS
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

func RunPagesProcessWithEnvs(t *testing.T, wait bool, pagesPath string, listeners []ListenSpec, promPort string, envs []string, extraArgs ...string) (teardown func()) {
	return runPagesProcess(t, wait, pagesPath, listeners, promPort, envs, extraArgs...)
}

func RunPagesProcessWithAuth(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string) func() {
	configFile, cleanup := defaultConfigFileWith(t,
		"auth-server=https://gitlab-auth.com",
		"auth-redirect-uri=https://projects.gitlab-example.com/auth")
	defer cleanup()

	return runPagesProcess(t, true, pagesPath, listeners, promPort, nil,
		"-config="+configFile,
	)
}

func RunPagesProcessWithAuthServer(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, authServer string) func() {
	return runPagesProcessWithAuthServer(t, pagesPath, listeners, promPort, nil, authServer)
}

func RunPagesProcessWithAuthServerWithSSLCertFile(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, authServer string) func() {
	return runPagesProcessWithAuthServer(t, pagesPath, listeners, promPort,
		[]string{"SSL_CERT_FILE=" + sslCertFile}, authServer)
}

func RunPagesProcessWithAuthServerWithSSLCertDir(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, sslCertFile string, authServer string) func() {
	// Create temporary cert dir
	sslCertDir, err := ioutil.TempDir("", "pages-test-SSL_CERT_DIR")
	require.NoError(t, err)

	// Copy sslCertFile into temp cert dir
	err = copyFile(sslCertDir+"/"+path.Base(sslCertFile), sslCertFile)
	require.NoError(t, err)

	innerCleanup := runPagesProcessWithAuthServer(t, pagesPath, listeners, promPort,
		[]string{"SSL_CERT_DIR=" + sslCertDir}, authServer)

	return func() {
		innerCleanup()
		os.RemoveAll(sslCertDir)
	}
}

func runPagesProcessWithAuthServer(t *testing.T, pagesPath string, listeners []ListenSpec, promPort string, extraEnv []string, authServer string) func() {
	configFile, cleanup := defaultConfigFileWith(t,
		"auth-server="+authServer,
		"auth-redirect-uri=https://projects.gitlab-example.com/auth")
	defer cleanup()

	return runPagesProcess(t, true, pagesPath, listeners, promPort, extraEnv,
		"-config="+configFile)
}

func runPagesProcess(t *testing.T, wait bool, pagesPath string, listeners []ListenSpec, promPort string, extraEnv []string, extraArgs ...string) (teardown func()) {
	t.Helper()

	_, err := os.Stat(pagesPath)
	require.NoError(t, err)

	args, tempfiles := getPagesArgs(t, listeners, promPort, extraArgs)
	cmd := exec.Command(pagesPath, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = &tWriter{t}
	cmd.Stderr = &tWriter{t}
	require.NoError(t, cmd.Start())
	t.Logf("Running %s %v", pagesPath, args)

	waitCh := make(chan struct{})
	go func() {
		cmd.Wait()
		for _, tempfile := range tempfiles {
			os.Remove(tempfile)
		}
		close(waitCh)
	}()

	cleanup := func() {
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

		if spec.Type == request.SchemeHTTPS {
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

func GetProxyRedirectPageWithCookie(t *testing.T, spec ListenSpec, host string, urlsuffix string, cookie string, https bool) (*http.Response, error) {
	schema := request.SchemeHTTP
	if https {
		schema = request.SchemeHTTPS
	}
	header := http.Header{
		"X-Forwarded-Proto": []string{schema},
		"X-Forwarded-Host":  []string{host},
		"cookie":            []string{cookie},
	}

	return GetRedirectPageWithHeaders(t, spec, host, urlsuffix, header)
}

func GetRedirectPageWithCookie(t *testing.T, spec ListenSpec, host, urlsuffix string, cookie string) (*http.Response, error) {
	return GetRedirectPageWithHeaders(t, spec, host, urlsuffix, http.Header{"cookie": []string{cookie}})
}

func GetRedirectPageWithHeaders(t *testing.T, spec ListenSpec, host, urlsuffix string, header http.Header) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header

	req.Host = host

	return TestHTTPSClient.Transport.RoundTrip(req)
}

func ClientWithConfig(tlsConfig *tls.Config) (*http.Client, func()) {
	tlsConfig.RootCAs = TestCertPool
	tr := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: tr}

	return client, tr.CloseIdleConnections
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

func NewGitlabDomainsSourceStub(t *testing.T, apiCalled *bool) *httptest.Server {
	*apiCalled = false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/internal/pages/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		*apiCalled = true
		domain := r.URL.Query().Get("host")
		path := "shared/lookups/" + domain + ".json"

		fixture, err := os.Open(path)
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)

			t.Logf("GitLab domain %s source stub served 204", domain)
			return
		}

		defer fixture.Close()
		require.NoError(t, err)

		_, err = io.Copy(w, fixture)
		require.NoError(t, err)

		t.Logf("GitLab domain %s source stub served lookup", domain)
	}
	mux.HandleFunc("/api/v4/internal/pages", handler)

	return httptest.NewServer(mux)
}

func newConfigFile(configs ...string) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "gitlab-pages-config")
	if err != nil {
		return "", err
	}
	defer f.Close()

	for _, config := range configs {
		_, err := fmt.Fprintf(f, "%s\n", config)
		if err != nil {
			return "", err
		}
	}

	return f.Name(), nil
}

func defaultConfigFileWith(t *testing.T, configs ...string) (string, func()) {
	configs = append(configs, "auth-client-id=clientID",
		"auth-client-secret=clientSecret",
		"auth-secret=authSecret")

	name, err := newConfigFile(configs...)
	require.NoError(t, err)

	cleanup := func() {
		err := os.Remove(name)
		require.NoError(t, err)
	}

	return name, cleanup
}

func copyFile(dest, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
