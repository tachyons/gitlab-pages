package acceptance_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
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
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
	"gitlab.com/gitlab-org/gitlab-pages/test/gitlabstub"
)

// The HTTPS certificate isn't signed by anyone. This http client is set up
// so it can talk to servers using it.
var (
	TestCertPool = x509.NewCertPool()

	// Proxyv2 will create a dummy request with src 10.1.1.1:1000
	// and dst 20.2.2.2:2000
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

func (l ListenSpec) Scheme() string {
	if l.Type == request.SchemeHTTPS || l.Type == "https-proxyv2" {
		return request.SchemeHTTPS
	}

	return request.SchemeHTTP
}

func (l ListenSpec) URL(suffix string) string {
	suffix = strings.TrimPrefix(suffix, "/")

	return fmt.Sprintf("%s://%s/%s", l.Scheme(), l.JoinHostPort(), suffix)
}

type dialContext func(ctx context.Context, network, addr string) (net.Conn, error)

func (l ListenSpec) proxyV2DialContext() dialContext {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer

		// bypass DNS resolution by going directly to host and port
		conn, err := d.DialContext(ctx, network, l.JoinHostPort())
		if err != nil {
			return nil, err
		}

		header := &proxyproto.Header{
			Version:           2,
			Command:           proxyproto.PROXY,
			TransportProtocol: proxyproto.TCPv4,
			SourceAddr: &net.TCPAddr{
				IP:   net.ParseIP("10.1.1.1"),
				Port: 1000,
			},
			DestinationAddr: &net.TCPAddr{
				IP:   net.ParseIP("20.2.2.2"),
				Port: 2000,
			},
		}

		_, err = header.WriteTo(conn)

		return conn, err
	}
}

func (l ListenSpec) httpsDialContext() dialContext {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer

		// bypass DNS resolution by going directly to host and port
		return d.DialContext(ctx, network, l.JoinHostPort())
	}
}

func (l ListenSpec) unixSocketDialContext() dialContext {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer

		return d.DialContext(ctx, "unix", l.Host)
	}
}

func (l ListenSpec) dialContext() dialContext {
	if l.Type == "https-proxyv2" {
		return l.proxyV2DialContext()
	}

	if l.Type == "unix" {
		return l.unixSocketDialContext()
	}

	return l.httpsDialContext()
}

func (l ListenSpec) Client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{RootCAs: TestCertPool},
			DialContext:           l.dialContext(),
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}
}

// Use a very short timeout to repeatedly check for the server to be up.
func (l ListenSpec) QuickTimeoutClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{RootCAs: TestCertPool},
			DialContext:           l.dialContext(),
			ResponseHeaderTimeout: 100 * time.Millisecond,
		},
	}
}

// Returns only once this spec points at a working TCP server
func (l ListenSpec) WaitUntilRequestSucceeds(done chan struct{}) error {
	timeout := 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return fmt.Errorf("server has shut down already")
		case <-ctx.Done():
			return fmt.Errorf("ctx done: %w for listener %v", ctx.Err(), l)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL("/@healthcheck"), nil)
			if err != nil {
				return err
			}

			response, err := l.QuickTimeoutClient().Transport.RoundTrip(req)

			if err == nil {
				response.Body.Close()

				if code := response.StatusCode; code >= 200 && code < 500 {
					return nil
				}
			}
		}
	}
}

func (l ListenSpec) JoinHostPort() string {
	if l.Type == "unix" {
		// The dialer ignores the addr parameter and uses
		// the socket path directly.
		// This is a stub used by ListenSpec#URL()
		// ListenSpec.Host cannot be used because it is
		// not a valid hostname.
		return "unix"
	}

	return net.JoinHostPort(l.Host, l.Port)
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

	source, err := gitlabstub.NewUnstartedServer(processCfg.gitlabStubOpts...)
	require.NoError(t, err)

	if source.TLS != nil {
		source.StartTLS()
	} else {
		source.Start()
	}

	gitLabAPISecretKey := CreateGitLabAPISecretKeyFixtureFile(t)
	processCfg.extraArgs = append(
		processCfg.extraArgs,
		"-pages-root", wd,
		"-internal-gitlab-server", source.URL,
		"-api-secret-key", gitLabAPISecretKey,
	)

	if processCfg.publicServer {
		processCfg.extraArgs = append(processCfg.extraArgs, "-gitlab-server", source.URL)
	}

	logBuf, cleanup := runPagesProcess(t, processCfg.wait, processCfg.pagesBinary, processCfg.listeners, "", processCfg.extraArgs...)

	t.Cleanup(func() {
		source.Close()
		chdirCleanup()
		cleanup()
	})

	return logBuf
}

func RunPagesProcessWithSSLCertFile(t *testing.T, listeners []ListenSpec, sslCertFile string) {
	t.Setenv("SSL_CERT_FILE", sslCertFile)

	RunPagesProcess(t,
		withListeners(listeners),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)
}

func RunPagesProcessWithSSLCertDir(t *testing.T, listeners []ListenSpec, sslCertFile string) {
	// Create temporary cert dir
	sslCertDir := t.TempDir()

	t.Setenv("SSL_CERT_DIR", sslCertDir)

	// Copy sslCertFile into temp cert dir
	err := copyFile(sslCertDir+"/"+path.Base(sslCertFile), sslCertFile)
	require.NoError(t, err)

	RunPagesProcess(t,
		withListeners(listeners),
		withArguments([]string{
			"-config=" + defaultAuthConfig(t),
		}),
	)
}

func runPagesProcess(t *testing.T, wait bool, pagesBinary string, listeners []ListenSpec, promPort string, extraArgs ...string) (*LogCaptureBuffer, func()) {
	t.Helper()

	_, err := os.Stat(pagesBinary)
	require.NoError(t, err)

	logBuf := &LogCaptureBuffer{}
	out := io.MultiWriter(&tWriter{t}, logBuf)

	args := getPagesArgs(t, listeners, promPort, extraArgs)
	cmd := exec.Command(pagesBinary, args...)
	cmd.Stdout = out
	cmd.Stderr = out
	require.NoError(t, cmd.Start())

	t.Logf("Running %s %v", pagesBinary, args)

	waitCh := make(chan struct{})
	go func() {
		require.NoError(t, cmd.Wait())
		close(waitCh)
	}()

	cleanup := func() {
		require.NoError(t, cmd.Process.Signal(os.Interrupt))
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

func getPagesArgs(t *testing.T, listeners []ListenSpec, promPort string, extraArgs []string) (args []string) {
	var hasHTTPS bool
	args = append(args, "-log-verbose=true")
	args = append(args, "-pages-status=/@healthcheck")

	for _, spec := range listeners {
		if spec.Type == "unix" {
			args = append(args, "-listen-http", spec.Host)
			continue
		}

		args = append(args, "-listen-"+spec.Type, spec.JoinHostPort())

		if strings.Contains(spec.Type, request.SchemeHTTPS) {
			hasHTTPS = true
		}
	}

	if hasHTTPS {
		key, cert := CreateHTTPSFixtureFiles(t)
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
	return GetPageFromListenerWithHeaders(t, spec, host, urlsuffix, http.Header{})
}

func GetPageFromListenerWithHeaders(t *testing.T, spec ListenSpec, host, urlSuffix string, header http.Header) (*http.Response, error) {
	t.Helper()

	url := spec.URL(urlSuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host
	req.Header = header

	return DoPagesRequest(t, spec, req)
}

func DoPagesRequest(t *testing.T, spec ListenSpec, req *http.Request) (*http.Response, error) {
	t.Logf("curl -X %s -H'Host: %s' %s", req.Method, req.Host, req.URL)

	return spec.Client().Do(req)
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

	return spec.Client().Transport.RoundTrip(req)
}

func ClientWithConfig(tlsConfig *tls.Config) (*http.Client, func()) {
	tlsConfig.RootCAs = TestCertPool
	tr := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: tr}

	return client, tr.CloseIdleConnections
}

func newConfigFile(t *testing.T, configs ...string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "gitlab-pages-config")
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

// RequireMetricEqual requests prometheus metrics and makes sure metric is there
func RequireMetricEqual(t *testing.T, metricsAddress, metricWithValue string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", metricsAddress))
	require.NoError(t, err)

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), metricWithValue)
}
