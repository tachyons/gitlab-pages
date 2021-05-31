package acceptance_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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
	"sync"
	"testing"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/test/acceptance/testdata"
)

// The HTTPS certificate isn't signed by anyone. This http client is set up
// so it can talk to servers using it.
var (
	// The HTTPS certificate isn't signed by anyone. This http client is set up
	// so it can talk to servers using it.
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

	// Proxyv2 client
	TestProxyv2Client = &http.Client{
		Transport: &http.Transport{
			DialContext:     Proxyv2DialContext,
			TLSClientConfig: &tls.Config{RootCAs: TestCertPool},
		},
	}

	QuickTimeoutProxyv2Client = &http.Client{
		Transport: &http.Transport{
			DialContext:           Proxyv2DialContext,
			TLSClientConfig:       &tls.Config{RootCAs: TestCertPool},
			ResponseHeaderTimeout: 100 * time.Millisecond,
		},
	}

	TestCertPool = x509.NewCertPool()

	// Proxyv2 will create a dummy request with src 10.1.1.1:1000
	// and dst 20.2.2.2:2000
	Proxyv2DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer

		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		header := &proxyproto.Header{
			Version:            2,
			Command:            proxyproto.PROXY,
			TransportProtocol:  proxyproto.TCPv4,
			SourceAddress:      net.ParseIP("10.1.1.1"),
			SourcePort:         1000,
			DestinationAddress: net.ParseIP("20.2.2.2"),
			DestinationPort:    2000,
		}

		_, err = header.WriteTo(conn)

		return conn, err
	}

	existingAcmeTokenPath    = "/.well-known/acme-challenge/existingtoken"
	notExistingAcmeTokenPath = "/.well-known/acme-challenge/notexistingtoken"
)

type tWriter struct {
	t *testing.T
}

func (t *tWriter) Write(b []byte) (int, error) {
	t.t.Log(string(bytes.TrimRight(b, "\r\n")))

	return len(b), nil
}

type LogCaptureBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *LogCaptureBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Read(p)
}
func (b *LogCaptureBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}
func (b *LogCaptureBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}
func (b *LogCaptureBuffer) Reset() {
	b.m.Lock()
	defer b.m.Unlock()

	b.b.Reset()
}

// ListenSpec is used to point at a gitlab-pages http server, preserving the
// type of port it is (http, https, proxy)
type ListenSpec struct {
	Type string
	Host string
	Port string
}

func supportedListeners() []ListenSpec {
	if !nettest.SupportsIPv6() {
		return ipv4Listeners
	}

	return listeners
}

func (l ListenSpec) URL(suffix string) string {
	scheme := request.SchemeHTTP
	if l.Type == request.SchemeHTTPS || l.Type == "https-proxyv2" {
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

		client := QuickTimeoutHTTPSClient
		if l.Type == "https-proxyv2" {
			client = QuickTimeoutProxyv2Client
		}

		response, err := client.Transport.RoundTrip(req)
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
func RunPagesProcess(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, true, pagesBinary, listeners, promPort, nil, extraArgs...)
	return cleanup
}

func RunPagesProcessWithoutWait(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, false, pagesBinary, listeners, promPort, nil, extraArgs...)
	return cleanup
}

func RunPagesProcessWithSSLCertFile(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, sslCertFile string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, true, pagesBinary, listeners, promPort, []string{"SSL_CERT_FILE=" + sslCertFile}, extraArgs...)
	return cleanup
}

func RunPagesProcessWithEnvs(t *testing.T, wait bool, pagesBinary string, listeners []ListenSpec, promPort string, envs []string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, wait, pagesBinary, listeners, promPort, envs, extraArgs...)
	return cleanup
}

func RunPagesProcessWithOutput(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, extraArgs ...string) (out *LogCaptureBuffer, teardown func()) {
	return runPagesProcess(t, true, pagesBinary, listeners, promPort, nil, extraArgs...)
}

func RunPagesProcessWithStubGitLabServer(t *testing.T, wait bool, pagesBinary string, listeners []ListenSpec, promPort string, envs []string, extraArgs ...string) (*LogCaptureBuffer, func()) {
	source := NewGitlabDomainsSourceStub(t, &stubOpts{})

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)
	pagesArgs := append([]string{"-gitlab-server", source.URL, "-api-secret-key", gitLabAPISecretKey, "-domain-config-source", "gitlab"}, extraArgs...)

	logBuf, cleanup := runPagesProcess(t, wait, pagesBinary, listeners, promPort, envs, pagesArgs...)

	return logBuf, func() {
		source.Close()
		cleanup()
	}
}

func RunPagesProcessWithAuth(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string) func() {
	configFile, cleanup := defaultConfigFileWith(t,
		"gitlab-server=https://gitlab-auth.com",
		"auth-redirect-uri=https://projects.gitlab-example.com/auth")
	defer cleanup()

	_, cleanup2 := runPagesProcess(t, true, pagesBinary, listeners, promPort, nil,
		"-config="+configFile,
	)
	return cleanup2
}

func RunPagesProcessWithGitlabServer(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, gitlabServer string) func() {
	return runPagesProcessWithGitlabServer(t, pagesBinary, listeners, promPort, nil, gitlabServer)
}

func RunPagesProcessWithGitlabServerWithSSLCertFile(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, sslCertFile string, gitlabServer string) func() {
	return runPagesProcessWithGitlabServer(t, pagesBinary, listeners, promPort,
		[]string{"SSL_CERT_FILE=" + sslCertFile}, gitlabServer)
}

func RunPagesProcessWithGitlabServerWithSSLCertDir(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, sslCertFile string, gitlabServer string) func() {
	// Create temporary cert dir
	sslCertDir, err := ioutil.TempDir("", "pages-test-SSL_CERT_DIR")
	require.NoError(t, err)

	// Copy sslCertFile into temp cert dir
	err = copyFile(sslCertDir+"/"+path.Base(sslCertFile), sslCertFile)
	require.NoError(t, err)

	innerCleanup := runPagesProcessWithGitlabServer(t, pagesBinary, listeners, promPort,
		[]string{"SSL_CERT_DIR=" + sslCertDir}, gitlabServer)

	return func() {
		innerCleanup()
		os.RemoveAll(sslCertDir)
	}
}

func runPagesProcessWithGitlabServer(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, extraEnv []string, gitlabServer string) func() {
	configFile, cleanup := defaultConfigFileWith(t,
		"gitlab-server="+gitlabServer,
		"auth-redirect-uri=https://projects.gitlab-example.com/auth")
	defer cleanup()

	_, cleanup2 := runPagesProcess(t, true, pagesBinary, listeners, promPort, extraEnv,
		"-config="+configFile)
	return cleanup2
}

func runPagesProcess(t *testing.T, wait bool, pagesBinary string, listeners []ListenSpec, promPort string, extraEnv []string, extraArgs ...string) (*LogCaptureBuffer, func()) {
	t.Helper()

	_, err := os.Stat(pagesBinary)
	require.NoError(t, err)

	logBuf := &LogCaptureBuffer{}
	out := io.MultiWriter(&tWriter{t}, logBuf)

	args, tempfiles := getPagesArgs(t, listeners, promPort, extraArgs)
	cmd := exec.Command(pagesBinary, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = out
	cmd.Stderr = out
	require.NoError(t, cmd.Start())
	t.Logf("Running %s %v", pagesBinary, args)

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

	return logBuf, cleanup
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

	if !contains(extraArgs, "-pages-root") {
		args = append(args, "-pages-root", "../../shared/pages")
	}

	// default resolver configuration to execute tests faster
	if !contains(extraArgs, "-gitlab-retrieval-") {
		args = append(args, "-gitlab-retrieval-timeout", "50ms",
			"-gitlab-retrieval-interval", "10ms",
			"-gitlab-retrieval-retries", "1")
	}

	if promPort != "" {
		args = append(args, "-metrics-address", promPort)
	}

	// most of our acceptance tests still work only with disk source
	// TODO: remove this with -domain-config-source flag itself:
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/571
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
	if !contains(extraArgs, "-domain-config-source") {
		args = append(args, "-domain-config-source", "disk")
	}

	args = append(args, getPagesDaemonArgs(t)...)
	args = append(args, extraArgs...)

	return
}

func contains(slice []string, s string) bool {
	for _, e := range slice {
		if strings.Contains(e, s) {
			return true
		}
	}
	return false
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

	return DoPagesRequest(t, spec, req)
}

func GetCompressedPageFromListener(t *testing.T, spec ListenSpec, host, urlsuffix string, encoding string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Host = host
	req.Header.Set("Accept-Encoding", encoding)

	return DoPagesRequest(t, spec, req)
}

func GetProxiedPageFromListener(t *testing.T, spec ListenSpec, host, xForwardedHost, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host
	req.Header.Set("X-Forwarded-Host", xForwardedHost)

	return DoPagesRequest(t, spec, req)
}

func DoPagesRequest(t *testing.T, spec ListenSpec, req *http.Request) (*http.Response, error) {
	t.Logf("curl -X %s -H'Host: %s' %s", req.Method, req.Host, req.URL)

	if spec.Type == "https-proxyv2" {
		return TestProxyv2Client.Do(req)
	}

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

	if spec.Type == "https-proxyv2" {
		return TestProxyv2Client.Transport.RoundTrip(req)
	}

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

			client := QuickTimeoutHTTPSClient
			if spec.Type == "https-proxyv2" {
				client = QuickTimeoutProxyv2Client
			}

			if response, err := client.Transport.RoundTrip(req); err == nil {
				nListening++
				response.Body.Close()
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	require.Equal(t, len(listeners), nListening, "all listeners must be accepting TCP connections")
}

type stubOpts struct {
	m                   sync.RWMutex
	apiCalled           bool
	statusReadyCount    int
	statusHandler       http.HandlerFunc
	pagesHandler        http.HandlerFunc
	pagesStatusResponse int
	pagesRoot           string
}

func NewGitlabDomainsSourceStub(t *testing.T, opts *stubOpts) *httptest.Server {
	t.Helper()
	require.NotNil(t, opts)

	currentStatusCount := 0

	mux := http.NewServeMux()
	statusHandler := func(w http.ResponseWriter, r *http.Request) {
		if currentStatusCount < opts.statusReadyCount {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}

	if opts.statusHandler != nil {
		statusHandler = opts.statusHandler
	}

	mux.HandleFunc("/api/v4/internal/pages/status", statusHandler)

	pagesHandler := defaultAPIHandler(t, opts)
	if opts.pagesHandler != nil {
		pagesHandler = opts.pagesHandler
	}

	mux.HandleFunc("/api/v4/internal/pages", pagesHandler)

	return httptest.NewServer(mux)
}

func (o *stubOpts) setAPICalled(v bool) {
	o.m.Lock()
	defer o.m.Unlock()

	o.apiCalled = v
}

func (o *stubOpts) getAPICalled() bool {
	o.m.RLock()
	defer o.m.RUnlock()

	return o.apiCalled
}

func lookupFromFile(t *testing.T, domain string, w http.ResponseWriter) {
	fixture, err := os.Open("../../shared/lookups/" + domain + ".json")
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

func defaultAPIHandler(t *testing.T, opts *stubOpts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("host")
		if domain == "127.0.0.1" {
			// shortcut for healthy checkup done by WaitUntilRequestSucceeds
			w.WriteHeader(http.StatusNoContent)
			return
		}

		opts.setAPICalled(true)

		if opts.pagesStatusResponse != 0 {
			w.WriteHeader(opts.pagesStatusResponse)
			return
		}

		// check if predefined response exists
		if responseFn, ok := testdata.DomainResponses[domain]; ok {
			err := json.NewEncoder(w).Encode(responseFn(opts.pagesRoot))
			require.NoError(t, err)
			return
		}

		// serve lookup from files
		lookupFromFile(t, domain, w)
	}
}

func newConfigFile(t *testing.T, configs ...string) string {
	t.Helper()

	f, err := ioutil.TempFile(os.TempDir(), "gitlab-pages-config")
	require.NoError(t, err)
	defer f.Close()

	for _, config := range configs {
		_, err := fmt.Fprintf(f, "%s\n", config)
		require.NoError(t, err)
	}

	return f.Name()
}

func defaultConfigFileWith(t *testing.T, configs ...string) (string, func()) {
	t.Helper()

	configs = append(configs, "auth-client-id=clientID",
		"auth-client-secret=clientSecret",
		"auth-secret=authSecret",
		"auth-scope=authScope",
	)

	name := newConfigFile(t, configs...)

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
