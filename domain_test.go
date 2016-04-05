package main

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
	"net/http/httptest"
	"github.com/stretchr/testify/require"
	"mime"
)

func TestGroupServeHTTP(t *testing.T) {
	setUpTests()

	testGroup := &domain{
		Group:   "group",
		Project: "",
	}

	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/", nil, "main-dir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/index.html", nil, "main-dir")
	assert.True(t, assert.HTTPRedirect(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project", nil))
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/", nil, "project-subdir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/index.html", nil, "project-subdir")
	assert.True(t, assert.HTTPRedirect(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/subdir", nil))
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/subdir/", nil, "project-subsubdir")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project2/", nil, "project2-main")
	assert.HTTPBodyContains(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project2/index.html", nil, "project2-main")
	assert.True(t, assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink", nil))
	assert.True(t, assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink/index.html", nil))
	assert.True(t, assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/symlink/subdir/", nil))
	assert.True(t, assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/project/fifo", nil))
	assert.True(t, assert.HTTPError(t, testGroup.ServeHTTP, "GET", "http://group.test.io/not-existing-file", nil))
}

func TestDomainServeHTTP(t *testing.T) {
	setUpTests()

	testDomain := &domain{
		Group:   "group",
		Project: "project2",
		Config: &domainConfig{
			Domain: "test.domain.com",
		},
	}

	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/index.html", nil, "project2-main")
	assert.HTTPRedirect(t, testDomain.ServeHTTP, "GET", "/subdir", nil)
	assert.HTTPBodyContains(t, testDomain.ServeHTTP, "GET", "/subdir/", nil, "project2-subdir")
	assert.HTTPError(t, testDomain.ServeHTTP, "GET", "/not-existing-file", nil)
}

func testHTTP404(t *testing.T, handler http.HandlerFunc, mode, url string, values url.Values, str interface{}) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(mode, url+"?"+values.Encode(), nil)
	require.NoError(t, err)
	handler(w, req)

	contentType, _, _ := mime.ParseMediaType(w.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusNotFound, w.Code, "HTTP status")
	assert.Equal(t, "text/html", contentType, "Content-Type")
	assert.Contains(t, w.Body.String(), str)
}

func TestGroup404ServeHTTP(t *testing.T) {
	setUpTests()

	testGroup := &domain{
		Group:   "group.404",
		Project: "",
	}

	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404/not/existing-file", nil, "Custom 404 project page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404/", nil, "Custom 404 project page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.no.404/not/existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/not/existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
	assert.HTTPBodyNotContains(t, testGroup.ServeHTTP, "GET", "http://group.404.test.io/project.404.symlink/not/existing-file", nil, "Custom 404 project page")
}

func TestDomain404ServeHTTP(t *testing.T) {
	setUpTests()

	testDomain := &domain{
		Group:   "group.404",
		Project: "domain.404",
		Config: &domainConfig{
			Domain: "domain.404.com",
		},
	}

	testHTTP404(t, testDomain.ServeHTTP, "GET", "http://group.404.test.io/not-existing-file", nil, "Custom 404 group page")
	testHTTP404(t, testDomain.ServeHTTP, "GET", "http://group.404.test.io/", nil, "Custom 404 group page")
}

func TestGroupCertificate(t *testing.T) {
	testGroup := &domain{
		Group:   "group",
		Project: "",
	}

	tls, err := testGroup.ensureCertificate()
	assert.Nil(t, tls)
	assert.Error(t, err)
}

func TestDomainNoCertificate(t *testing.T) {
	testDomain := &domain{
		Group:   "group",
		Project: "project2",
		Config: &domainConfig{
			Domain: "test.domain.com",
		},
	}

	tls, err := testDomain.ensureCertificate()
	assert.Nil(t, tls)
	assert.Error(t, err)

	_, err2 := testDomain.ensureCertificate()
	assert.Error(t, err)
	assert.Equal(t, err, err2)
}

func TestDomainCertificate(t *testing.T) {
	testDomain := &domain{
		Group:   "group",
		Project: "project2",
		Config: &domainConfig{
			Domain: "test.domain.com",
			Certificate: `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`,
			Key: `-----BEGIN PRIVATE KEY-----
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
-----END PRIVATE KEY-----`,
		},
	}

	tls, err := testDomain.ensureCertificate()
	assert.NotNil(t, tls)
	assert.NoError(t, err)
}
