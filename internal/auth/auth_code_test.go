package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEncryptAndDecryptSignedCode(t *testing.T) {
	auth := createTestAuth(t, "", "")

	tests := map[string]struct {
		auth              *Auth
		encDomain         string
		code              string
		expectedEncErrMsg string
		decDomain         string
		expectedDecErrMsg string
	}{
		"happy_path": {
			auth:      auth,
			encDomain: "domain",
			decDomain: "domain",
			code:      "code",
		},
		"empty_domain": {
			auth:              auth,
			encDomain:         "",
			code:              "code",
			expectedEncErrMsg: "empty domain or code",
		},
		"empty_code": {
			auth:              auth,
			encDomain:         "domain",
			code:              "",
			expectedEncErrMsg: "empty domain or code",
		},
		"different_dec_domain": {
			auth:              auth,
			encDomain:         "domain",
			decDomain:         "another",
			code:              "code",
			expectedDecErrMsg: "cipher: message authentication failed",
		},
		"expired_token": {
			auth: func() *Auth {
				newAuth := *auth
				newAuth.jwtExpiry = time.Nanosecond
				newAuth.now = func() time.Time {
					return time.Time{}
				}

				return &newAuth
			}(),
			encDomain:         "domain",
			code:              "code",
			decDomain:         "domain",
			expectedDecErrMsg: "Token is expired",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encCode, err := test.auth.EncryptAndSignCode(test.encDomain, test.code)
			if test.expectedEncErrMsg != "" {
				require.EqualError(t, err, test.expectedEncErrMsg)
				require.Empty(t, encCode)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, encCode)

			decCode, err := test.auth.DecryptCode(encCode, test.decDomain)
			if test.expectedDecErrMsg != "" {
				require.EqualError(t, err, test.expectedDecErrMsg)
				require.Empty(t, decCode)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.code, decCode)
		})
	}
}

func TestDecryptCodeWithInvalidJWT(t *testing.T) {
	auth1 := createTestAuth(t, "", "")
	auth2 := createTestAuth(t, "", "")
	auth2.jwtSigningKey = []byte("another signing key")

	encCode, err := auth1.EncryptAndSignCode("domain", "code")
	require.NoError(t, err)

	decCode, err := auth2.DecryptCode(encCode, "domain")
	require.EqualError(t, err, "signature is invalid")
	require.Empty(t, decCode)
}
