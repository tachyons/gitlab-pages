package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		log.Println("Chdir:", err)
	} else {
		chdirSet = true
	}
}

// The HTTPS certificate isn't signed by anyone. This http client is set up
// so it can talk to servers using it.
var InsecureHTTPSClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var CertificateFixture = `-----BEGIN CERTIFICATE-----
MIICWDCCAcGgAwIBAgIJAMyzCfoGEwVNMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTYwMjExMTcxNzM2WhcNMjYwMjA4MTcxNzM2WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQC2ZSzGIlv2zRsELkmEA1JcvIdsFv80b0NbBftewDAQRuyPlhGNifFx6v7+3O1F
5+f+So43N0QbdrHu11K+ZuXNc6hUy0ofG/eRqXniGZEn8paUdQ98sWsbWelBDNeg
WX4FQomynjyxbG+3IuJR5UHoLWhrJ9+pbPrT915eObbaTQIDAQABo1AwTjAdBgNV
HQ4EFgQUGAhDu+gfckg4IkHRCQWBn4ltKV4wHwYDVR0jBBgwFoAUGAhDu+gfckg4
IkHRCQWBn4ltKV4wDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOBgQAaGx5U
JRW5HC9dXADLf9OnmJRqWi3VNXEXWFk5XgHKc1z7KIBWMsdj+1gzm5ltRO7hkHw9
bx6jQKZBRiUxyqTFw9Ywrk1fYFAxk8hxuqVYcGdonImJarHZTdVMBRWut9+EZBVm
77eYbz2zASNpy++QIg85YgQum9uqREClHRBsxQ==
-----END CERTIFICATE-----`

var KeyFixture = `-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBALZlLMYiW/bNGwQu
SYQDUly8h2wW/zRvQ1sF+17AMBBG7I+WEY2J8XHq/v7c7UXn5/5Kjjc3RBt2se7X
Ur5m5c1zqFTLSh8b95GpeeIZkSfylpR1D3yxaxtZ6UEM16BZfgVCibKePLFsb7ci
4lHlQegtaGsn36ls+tP3Xl45ttpNAgMBAAECgYAAqZFmDs3isY/9jeV6c0CjUZP0
UokOubC27eihyXTjOj61rsfVicC0tzPB3S+HZ3YyODcYAD1hFCdFRMbqJhmDiewK
5GfATdNQeNARCfJdjYn57NKaXm7rc4C3so1YfxTL6k9QGJgTcybXiClQPDrhkZt3
YLIeeJbY3OppLqjzgQJBAN5AzwyUqX5eQIUncQKcFY0PIjfFTku62brT7hq+TlqY
1B6n3GUtIX+tyYg1qusy4KUUSzMslXJubHsxKanGqZ0CQQDSFwzK7KEYoZol5OMX
mRsavc3iXmmEkkNRdNb1R4UqrlasPeeIeO1CfoD2RPcQhZCwFtR8xS8u6X9ncfC4
qyxxAkAhpQvy6ppR7/Cyd4sLCxfUF8NlT/APVMTbHHQCBmcUHeiWj3C0vEVC78r/
XKh4HGaXdt//ajNhdEflykZ1VgadAkB6Zh934mEA3rXWOgHsb7EQ5WAb8HF9YVGD
FZVfFaoJ8cRhWTeZlQp14Qn1cLyYjZh8XvCxOJiCtlsZw5JBpMihAkBA6ltWb+aZ
EBjC8ZRwZE+cAzmxaYPSs2J7JhS7X7H7Ax7ShhvHI4br3nqf00H4LkvtcHkn5d9G
MwE1w2r4Deww
-----END PRIVATE KEY-----`

func CreateHTTPSFixtureFiles(t *testing.T) (key string, cert string) {
	keyfile, err := ioutil.TempFile("", "https-fixture")
	assert.NoError(t, err)
	key = keyfile.Name()
	keyfile.Close()

	certfile, err := ioutil.TempFile("", "https-fixture")
	assert.NoError(t, err)
	cert = certfile.Name()
	certfile.Close()

	assert.NoError(t, ioutil.WriteFile(key, []byte(KeyFixture), 0644))
	assert.NoError(t, ioutil.WriteFile(cert, []byte(CertificateFixture), 0644))

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

	return fmt.Sprintf("%s://%s/%s", scheme, l.JoinHostPort(), suffix)
}

// Returns only once this spec points at a working TCP server
func (l ListenSpec) WaitUntilListening() {
	for {
		conn, _ := net.Dial("tcp", l.JoinHostPort())
		if conn != nil {
			conn.Close()
			break
		}
	}
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
	_, err := os.Stat(pagesPath)
	assert.NoError(t, err)

	args, tempfiles := getPagesArgs(t, listeners, promPort, extraArgs)
	cmd := exec.Command(pagesPath, args...)
	cmd.Stdout = &tWriter{t}
	cmd.Stderr = &tWriter{t}
	cmd.Start()
	t.Logf("Running %s %v", pagesPath, args)

	// Wait for all TCP servers to be open. Even with this, gitlab-pages
	// will sometimes return 404 if a HTTP request comes in before it has
	// updated its set of domains. This usually takes < 1ms, hence the sleep
	// for now. Without it, intermittent failures occur.
	//
	// TODO: replace this with explicit status from the pages binary
	// TODO: fix the first-request race
	for _, spec := range listeners {
		spec.WaitUntilListening()
	}
	time.Sleep(500 * time.Millisecond)

	return func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Process.Wait()
		for _, tempfile := range tempfiles {
			os.Remove(tempfile)
		}
	}
}

func getPagesArgs(t *testing.T, listeners []ListenSpec, promPort string, extraArgs []string) (args, tempfiles []string) {
	var hasHTTPS bool

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
	if os.Geteuid() == 0 {
		t.Log("Running pages as a daemon")
		args = append(args, "-daemon-uid", "0")
		args = append(args, "-daemon-gid", "65534") // Root user can switch to "nobody"
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
