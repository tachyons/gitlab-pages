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
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type tWriter struct {
	t *testing.T
}

func (t *tWriter) Write(b []byte) (int, error) {
	t.t.Log(string(bytes.TrimRight(b, "\r\n")))

	return len(b), nil
}

var chdirSet = false

func setUpTests() {
	if chdirSet {
		return
	}

	err := os.Chdir("shared/pages")
	if err != nil {
		log.WithError(err).Print("chdir")
	} else {
		chdirSet = true
	}
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

	CertificateFixture = `-----BEGIN CERTIFICATE-----
MIIDZDCCAkygAwIBAgIRAOtN9/zy+gFjdsgpKq3QRdQwDQYJKoZIhvcNAQELBQAw
MzEUMBIGA1UEChMLTG9nIENvdXJpZXIxGzAZBgNVBAMTEmdpdGxhYi1leGFtcGxl
LmNvbTAgFw0xODAzMjMxODMwMDZaGA8yMTE4MDIyNzE4MzAwNlowMzEUMBIGA1UE
ChMLTG9nIENvdXJpZXIxGzAZBgNVBAMTEmdpdGxhYi1leGFtcGxlLmNvbTCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKULsxpnazXX5RsVzayrAQB+lWwr
Wef5L5eDhSsIsBLbelYp5YB4TmVRt5x7bWKOOJSBsOfwHZHKJXdu+uuX2RenZlhk
3Qpq9XGaPZjYm/NHi8gBHPAtz5sG5VaKNvkfTzRGnO9CWA9TM1XtYiOBq94dO+H3
c+5jP5Yw+mJ+hA+i2058zF8nRlUHArEno2ofrHwE0LMZ11VskpXtWnVfs3voLs8p
r76KXPBFkMJR4qkWrMDF5Y5MbsQ0zisn6KXrTyV0S4MQh4vSyPdFHnEzvJ07rm5x
4RTWrjgQeQ2DjZjQvRmaDzlVBK9kaMkJ1Si3agK+gpji6d6WZ/Mb2el1GK8CAwEA
AaNxMG8wDgYDVR0PAQH/BAQDAgKkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1Ud
EwEB/wQFMAMBAf8wNwYDVR0RBDAwLoIUKi5naXRsYWItZXhhbXBsZS5jb22HBH8A
AAGHEAAAAAAAAAAAAAAAAAAAAAEwDQYJKoZIhvcNAQELBQADggEBAJ0NM8apK0xI
YxMstP/dCQXtR0wyREGSD/eOpeY3bWlqCbpRgMFUGjQlrsEozcPZOCSCKX5p+tym
7GsnYtXkwbsuURoSz+5IlhRPVHcUlUeGRdv3/gCd8fDXiigALCsB6GrkMG5cUfh+
x5p52AC3eQdWTDoxNou+2gzwkAl8iJc13Ykusst0YUqcsXKqTuei2quxFv0pEBSO
p8wEixoicLFNqPnIDmgx5894DAn0bccNXgRWtq8lLbdhGUlBbpatevvFMgNvFUbe
eeGb9D0EfpxmzxUl+L0xZtfg3f7cu5AgLG8tb6l4AK6NPVuXN8DmUgvnauWJjZME
fgStI+IRNVg=
-----END CERTIFICATE-----
`

	KeyFixture = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEApQuzGmdrNdflGxXNrKsBAH6VbCtZ5/kvl4OFKwiwEtt6Vinl
gHhOZVG3nHttYo44lIGw5/Adkcold27665fZF6dmWGTdCmr1cZo9mNib80eLyAEc
8C3PmwblVoo2+R9PNEac70JYD1MzVe1iI4Gr3h074fdz7mM/ljD6Yn6ED6LbTnzM
XydGVQcCsSejah+sfATQsxnXVWySle1adV+ze+guzymvvopc8EWQwlHiqRaswMXl
jkxuxDTOKyfopetPJXRLgxCHi9LI90UecTO8nTuubnHhFNauOBB5DYONmNC9GZoP
OVUEr2RoyQnVKLdqAr6CmOLp3pZn8xvZ6XUYrwIDAQABAoIBAHhP5QnUZeTkMtDh
vgKmzZ4sqIQnvexKTBUo/MR4GtJESBPTisdx68QUI8LgfsafYkNvnyQUd5m1QEam
Eif3k3uYvhSlwjQ78BwWEdz/2f8oIo9zsEKtQm+CQWAqdRR5bGVxLCmFtWfGgN+c
ojO77SuHKAX7OvmGQ+4aWgu+qkoyg/chIpPXMduAjLMtN3eg60ZqJ5KrKuIF63Bb
xkPQvzJueB9SfUurmKjUltDMx6G/9RZyS0OIRGyL9Qp8MZ8jE23cXOcDgm0HhkPq
W4LU++aWAOLYziTjnhjJ+4Iz9R7U8sCmk1wgnK/tapVcJf41R98WuGluyjXpsXgA
k7vmofECgYEAzuGun9lZ7xGwPifp6vGanWXnW+JiZgCTGuHWgQLIXWYcLfoI3kpH
eLBYINBwvjIQ7P6UxsGSSXd+T82t+8W2LLc2fiKFKE1LVySpH99+cfmIPXxrviOz
GBX9LTdSCdGkgb54m8aJCpNFnKw5wYgcW1L8CaXXly2Z/KNrGR9R/YUCgYEAzDs4
19HqlutGLTC30/ziiiIfDaBbX9AzBdUfp9GdT53Mi/7bfxpW/sL4RjG2fGgmN6ua
fh5npT9AB1ldcEg2qfyOJPt1Ubdi6ek9lx8AB2RMhwdihgX+7bjVMFtjg4b8z5C1
jQbEr1rhFdpaGyNehtAXDgCbDWQBYnBrmM0rCaMCgYBip1Qyfd9ZFcJJoZb2pofo
jvOo6Weq5JNBungjxUPu5gaCFj2sYxd6Af3EiCF7UTypBy3DKgOsbQMa4yYYbcvV
vviJZcTB1zoaMC1GObl+eFPzniVy4mtBDRtSOJMyg3pDNKUnA6HOHTSQ5cAU/ecn
1YbCwwbv3JsV0of7zue2UQKBgQCVc0j3dd9rLSQfcaUz9bx5RNrgh9YV2S9dN0aA
8f1iA6FpWMiazFWY/GfeRga6JyTAXE0juXAzFoPuXNDpl46Y+f2yxmhlsgMqFMpD
SiYlQppVvWu1k7GnmDg5uMarux5JbiXM24UWpTRNX4nMjidgE+qrDnpoZCQ3Ovkh
yhGSbQKBgD3VEnPiSUmXBo39kPcnPg93E3JfdAOiOwIB2qwfYzg9kpmuTWws+DFz
lKpMI27YkmnPqROQ2NTUfdxYmw3EHHMAsvnmHeMNGn3ijSUZVKmPfV436Qc8iVci
s4wKoCRhBUZ52sHki/ieb+5hycT3JnVXMDtbJxgXFW5a86usXEpO
-----END RSA PRIVATE KEY-----`

	TestCertPool = x509.NewCertPool()
)

func init() {
	if ok := TestCertPool.AppendCertsFromPEM([]byte(CertificateFixture)); !ok {
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

	require.NoError(t, ioutil.WriteFile(key, []byte(KeyFixture), 0644))
	require.NoError(t, ioutil.WriteFile(cert, []byte(CertificateFixture), 0644))

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

func runPagesProcess(t *testing.T, wait bool, pagesPath string, listeners []ListenSpec, promPort string, extraEnv []string, extraArgs ...string) (teardown func()) {
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

	// At least one of `-daemon-uid` and `-daemon-gid` must be non-zero
	if daemon, _ := strconv.ParseBool(os.Getenv("TEST_DAEMONIZE")); daemon {
		if os.Geteuid() == 0 {
			t.Log("Running pages as a daemon")
			args = append(args, "-daemon-uid", "0")
			args = append(args, "-daemon-gid", "65534") // Root user can switch to "nobody"
		} else {
			t.Log("Privilege-dropping requested but not running as root!")
			t.FailNow()
		}
	}

	args = append(args, extraArgs...)

	return
}

// Does a HTTP(S) GET against the listener specified, setting a fake
// Host: and constructing the URL from the listener and the URL suffix.
func GetPageFromListener(t *testing.T, spec ListenSpec, host, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host

	return DoPagesRequest(t, req)
}

func DoPagesRequest(t *testing.T, req *http.Request) (*http.Response, error) {
	t.Logf("curl -X %s -H'Host: %s' %s", req.Method, req.Host, req.URL)

	return TestHTTPSClient.Do(req)
}

func GetRedirectPage(t *testing.T, spec ListenSpec, host, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
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
