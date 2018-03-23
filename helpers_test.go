package main

import (
	"bytes"
	"crypto/tls"
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
var InsecureHTTPSClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 100 * time.Millisecond,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	},
}

var CertificateFixture = `-----BEGIN CERTIFICATE-----
MIIDPDCCAiSgAwIBAgIRAJxeIG2dasNCFzigvI3rSSowDQYJKoZIhvcNAQELBQAw
MzEUMBIGA1UEChMLTG9nIENvdXJpZXIxGzAZBgNVBAMTEmdpdGxhYi1leGFtcGxl
LmNvbTAgFw0xODAzMjIxOTE5MjZaGA8yMTE4MDIyNjE5MTkyNlowMzEUMBIGA1UE
ChMLTG9nIENvdXJpZXIxGzAZBgNVBAMTEmdpdGxhYi1leGFtcGxlLmNvbTCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKHXQX7TsNTybojzmSzCwC8Hgk21
VjIZT0aGZAGaQXL9npYq3ic+hIuWO8xid5KoTQJV4SNS+kB5nr4kTfrRbGVo7RWF
P1HZ5TZoeWPngyz82eYGaiLan4oSzE5wvPcHk90/CLeO/OeILy9w6Q+Ns9vR87RZ
iaVMivi6MWT/kRGy9KzvKFQKxxfReXAqoKyUk+SSP9vJ5ujX0vvIye9fn0glN2oM
nR/M4LjXNNJiV+J5rYsek8DL5PrRWWChMP+I+JFhUc4aVI/aqkBCnluxIamS5iLt
035Q7laqfOKrB3/SI9AEQm5XrYtUBH0LtFOphzXVR1hYeDHr8Df1gBM6YjECAwEA
AaNJMEcwDgYDVR0PAQH/BAQDAgKkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1Ud
EwEB/wQFMAMBAf8wDwYDVR0RBAgwBocEfwAAATANBgkqhkiG9w0BAQsFAAOCAQEA
RbJQf+dpgSGnCgHzX0bmESo2RUghFdsZ9RmLOqcIFEPaMLAwUyPsI2UL1bSv9FtW
BVIOgmNUQexOgJ3rIpKUp3Nbbr7QXDaoyC2teL6NMiYuIM3czX7zU5vhTduLpWEF
yPSC+5jLksFayhNTDmZHc8jcpuTLBg48iPQQjy84jfCv0PVvQ7TuXYRVgMb7PuHo
aqH4xpoFHutMUSuIo1naiHjw8wwC+UvFuS1FUowLxWzreOW43vp026SGeoCldKYY
p+e6LzsqwyIK3BuWJ+2cH4UyCt8Dp758sNZHDoBLKMx8ZpA+Y1WzVohLxk5yEQkl
QXUumHMXqybXNEi7PPsznw==
-----END CERTIFICATE-----`

var KeyFixture = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAoddBftOw1PJuiPOZLMLALweCTbVWMhlPRoZkAZpBcv2elire
Jz6Ei5Y7zGJ3kqhNAlXhI1L6QHmeviRN+tFsZWjtFYU/UdnlNmh5Y+eDLPzZ5gZq
ItqfihLMTnC89weT3T8It47854gvL3DpD42z29HztFmJpUyK+LoxZP+REbL0rO8o
VArHF9F5cCqgrJST5JI/28nm6NfS+8jJ71+fSCU3agydH8zguNc00mJX4nmtix6T
wMvk+tFZYKEw/4j4kWFRzhpUj9qqQEKeW7EhqZLmIu3TflDuVqp84qsHf9Ij0ARC
bleti1QEfQu0U6mHNdVHWFh4MevwN/WAEzpiMQIDAQABAoIBABK3TQi4vHtz6dqG
qVEm2IjXynboIKa8jJFwW0JgL2936w4cuQI61aM65YF2ZbOdKQK7IcUvBGfOaNA+
bJI0A+AaaUiS10bE9x/6pwcpr97VAvH6De4n8ElMcTolCYVb5/qvHnfz3kV8V1Ca
MymsTn9+YTubGzL1jiDDj5DJiWJNa6XqJqF9eh4B7nxnrjO8T3NMTI3lvyg/Nkrx
6l0qhEG+Eu1Gdzv8t1mTb4wcz1lpC152oMtFZqgWEjMHjZryVgjPq8t25b8OhZk3
e8sYX0JcqHZl/zqVlLoxQbSmH8/ePLH1Si5RFMxxQKhRgp2I068Us15rt2k0PnMh
C6y1w1ECgYEAw1b8lLtvtm4NrBcqD0THYs+35ua2B23MxoksasvTlpG7Je3HSf6M
tOIcv32cLjh4/Q1lnzdrO3lOzzDVHcKRXuUSI9C7CUFhnAKLY/0d255RjG7Hm+vv
OyRXJYIli6+m/fzu/97Eyjs08DO+Rg/ONu+UqWlusSvEwl93Z2u7hn0CgYEA1Bkw
RQqrOVlFdRv+jfraBVpO2enRzWBHZA+0AWdGZ3vMkyVHxfVOyRjfPOd1N1YnTfqH
1X+b+lpWULpLH/SVeidWSUcEhtuew1TRGexmz3XCN7i6PiwtXjhOAgY9YwVMiOMy
CKVIrL6bJAqJwniRiTn6aXj+L0xJcPL1GemMtMUCgYEAsPGJyJxk3CaioeE1yzDt
P5eTKUiRWPdgB/NX1cGef4SwtvHFlURMZslvaxI4ODIVfnv1Mp07uFrxRYMheVy2
2/O6U9EOq5qa9XvkkgVFV5v4mLH8hEPap4MKocJbikXpiablQ8eiEOJC2Na2I7bL
gD3TNwZ3K2vPRpa9jWQsMO0CgYEAy3oSxdmzZIRRT0V5E4raCJKX3RUlcstwEf3C
qioC8Bpjq7LzRWXOnLxgxlQjLuBXOscj813GLQrnjfD7S3/gu1zruccI/7vIdwpy
xFT4WQVXOw/clPLa325S4DxOPiYCQ7z67jJrI1aFDbGSceArdyQJKZCrAoNEXbio
DaDynSUCgYBCaM4rHpfkpCgOCtZg+hbwrmYbpRiZ6LJRi5t8M5c5ERUh8rlvADBv
S3Tg9fq/TkV8IZKsIjc2Rgs49+/XdlNZdUE59Z/t/OzXv8DylGt5E0YH4kN8qd6e
zTa+zLrR664UL7KDXSZuY+kHfsQQwxvsGcQma7ig1PUjlPhKLfYrRQ==
-----END RSA PRIVATE KEY-----`

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

		response, err := InsecureHTTPSClient.Transport.RoundTrip(req)
		if err != nil {
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

// Does an insecure HTTP GET against the listener specified, setting a fake
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

	return InsecureHTTPSClient.Do(req)
}

func GetRedirectPage(t *testing.T, spec ListenSpec, host, urlsuffix string) (*http.Response, error) {
	url := spec.URL(urlsuffix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = host

	return InsecureHTTPSClient.Transport.RoundTrip(req)
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

			if response, err := InsecureHTTPSClient.Transport.RoundTrip(req); err == nil {
				nListening++
				response.Body.Close()
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	require.Equal(t, len(listeners), nListening, "all listeners must be accepting TCP connections")
}
