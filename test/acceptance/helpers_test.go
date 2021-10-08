package acceptance_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

	"github.com/gorilla/mux"
	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
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

// RunPagesProcessWithoutGitLabStub will start a gitlab-pages process with the specified listeners
// and return a function you can call to shut it down again. Use
// GetPageFromProcess to do a HTTP GET against a listener.
//
// If run as root via sudo, the gitlab-pages process will drop privileges
func RunPagesProcessWithoutGitLabStub(t *testing.T, pagesBinary string, listeners []ListenSpec, promPort string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, true, pagesBinary, listeners, promPort, nil, extraArgs...)
	return cleanup
}

func RunPagesProcessWithEnvs(t *testing.T, wait bool, pagesBinary string, listeners []ListenSpec, promPort string, envs []string, extraArgs ...string) (teardown func()) {
	_, cleanup := runPagesProcess(t, wait, pagesBinary, listeners, promPort, envs, extraArgs...)
	return cleanup
}

func RunPagesProcess(t *testing.T, opts ...processOption) *LogCaptureBuffer {
	chdir := false
	chdirCleanup := testhelpers.ChdirInPath(t, "../../shared/pages", &chdir)

	wd, err := os.Getwd()
	require.NoError(t, err)

	processCfg := defaultProcessConfig

	for _, opt := range opts {
		opt(&processCfg)
	}

	if processCfg.gitlabStubOpts.pagesRoot == "" {
		processCfg.gitlabStubOpts.pagesRoot = wd
	}

	source := NewGitlabDomainsSourceStub(t, processCfg.gitlabStubOpts)

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)
	processCfg.extraArgs = append(
		processCfg.extraArgs,
		"-pages-root", wd,
		"-internal-gitlab-server", source.URL,
		"-api-secret-key", gitLabAPISecretKey,
	)

	logBuf, cleanup := runPagesProcess(t, processCfg.wait, processCfg.pagesBinary, processCfg.listeners, "", processCfg.envs, processCfg.extraArgs...)

	t.Cleanup(func() {
		source.Close()
		chdirCleanup()
		cleanup()
	})

	return logBuf
}

func RunPagesProcessWithSSLCertFile(t *testing.T, listeners []ListenSpec, sslCertFile string) {
	runPagesWithAuthAndEnv(t, listeners, []string{"SSL_CERT_FILE=" + sslCertFile})
}

func RunPagesProcessWithSSLCertDir(t *testing.T, listeners []ListenSpec, sslCertFile string) {
	// Create temporary cert dir
	sslCertDir, err := os.MkdirTemp("", "pages-test-SSL_CERT_DIR")
	require.NoError(t, err)

	// Copy sslCertFile into temp cert dir
	err = copyFile(sslCertDir+"/"+path.Base(sslCertFile), sslCertFile)
	require.NoError(t, err)

	runPagesWithAuthAndEnv(t, listeners, []string{"SSL_CERT_DIR=" + sslCertDir})

	t.Cleanup(func() {
		os.RemoveAll(sslCertDir)
	})
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

		if strings.Contains(spec.Type, request.SchemeHTTPS) {
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

func GetPageFromListenerWithRemoteAddrAndXFF(t *testing.T, spec ListenSpec, host, urlsuffix, xForwardedFor, xForwardedHost string) (*http.Response, error) {
	t.Helper()

	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host
	req.Header.Set("X-Forwarded-For", xForwardedFor)
	req.Header.Set("X-Forwarded-Host", xForwardedHost)

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

type stubOpts struct {
	m                   sync.RWMutex
	apiCalled           bool
	authHandler         http.HandlerFunc
	userHandler         http.HandlerFunc
	pagesHandler        http.HandlerFunc
	pagesStatusResponse int
	pagesRoot           string
}

func NewGitlabDomainsSourceStub(t *testing.T, opts *stubOpts) *httptest.Server {
	t.Helper()
	require.NotNil(t, opts)

	router := mux.NewRouter()

	pagesHandler := defaultAPIHandler(t, opts)
	if opts.pagesHandler != nil {
		pagesHandler = opts.pagesHandler
	}

	router.HandleFunc("/api/v4/internal/pages", pagesHandler)

	authHandler := defaultAuthHandler(t, opts)
	if opts.authHandler != nil {
		authHandler = opts.authHandler
	}

	router.HandleFunc("/oauth/token", authHandler)

	userHandler := defaultUserHandler(t, opts)
	if opts.userHandler != nil {
		userHandler = opts.userHandler
	}

	router.HandleFunc("/api/v4/user", userHandler)

	router.HandleFunc("/api/v4/projects/{project_id:[0-9]+}/pages_access", func(w http.ResponseWriter, r *http.Request) {
		if handleAccessControlArtifactRequests(t, w, r) {
			return
		}

		handleAccessControlRequests(t, w, r)
	})

	return httptest.NewServer(router)
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
	if errors.Is(err, fs.ErrNotExist) {
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
			err := json.NewEncoder(w).Encode(responseFn(t, opts.pagesRoot))
			require.NoError(t, err)
			return
		}

		// serve lookup from files
		lookupFromFile(t, domain, w)
	}
}

func defaultAuthHandler(t *testing.T, opts *stubOpts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		err := json.NewEncoder(w).Encode(struct {
			AccessToken string `json:"access_token"`
		}{
			AccessToken: "abc",
		})
		require.NoError(t, err)
	}
}

func defaultUserHandler(t *testing.T, opts *stubOpts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}
}

func newConfigFile(t *testing.T, configs ...string) string {
	t.Helper()

	f, err := os.CreateTemp(os.TempDir(), "gitlab-pages-config")
	require.NoError(t, err)
	defer f.Close()

	for _, config := range configs {
		_, err := fmt.Fprintf(f, "%s\n", config)
		require.NoError(t, err)
	}

	return f.Name()
}

func defaultConfigFileWith(t *testing.T, configs ...string) string {
	t.Helper()

	configs = append(configs, "auth-client-id=clientID",
		"auth-client-secret=clientSecret",
		"auth-secret=authSecret",
		"auth-scope=authScope",
	)

	name := newConfigFile(t, configs...)

	t.Cleanup(func() {
		err := os.Remove(name)
		require.NoError(t, err)
	})

	return name
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

func setupTransport(t *testing.T) {
	t.Helper()

	transport := (TestHTTPSClient.Transport).(*http.Transport)
	defer func(t time.Duration) {
		transport.ResponseHeaderTimeout = t
	}(transport.ResponseHeaderTimeout)
	transport.ResponseHeaderTimeout = 5 * time.Second
}
