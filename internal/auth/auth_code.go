package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/hkdf"
)

var (
	errInvalidToken      = errors.New("invalid token")
	errEmptyDomainOrCode = errors.New("empty domain or code")
	errInvalidNonce      = errors.New("invalid nonce")
	errInvalidCode       = errors.New("invalid code")
)

// EncryptAndSignCode encrypts the OAuth code deriving the key from the domain.
// It adds the code and domain as JWT token claims and signs it using signingKey derived from
// the Auth secret.
func (a *Auth) EncryptAndSignCode(domain, code string) (string, error) {
	if domain == "" || code == "" {
		return "", errEmptyDomainOrCode
	}

	// for FIPS mode, the nonce size has to be equal to the gcmStandardNonceSize i.e. 12
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/726
	nonce := securecookie.GenerateRandomKey(12)
	if nonce == nil {
		// https://github.com/gorilla/securecookie/blob/f37875ef1fb538320ab97fc6c9927d94c280ed5b/securecookie.go#L513
		return "", errInvalidNonce
	}

	aesGcm, err := a.newAesGcmCipher(domain, nonce)
	if err != nil {
		return "", err
	}

	// encrypt code with a randomly generated nonce
	encryptedCode := aesGcm.Seal(nil, nonce, []byte(code), nil)

	// generate JWT token claims with encrypted code
	claims := jwt.MapClaims{
		// standard claims
		"iss": "gitlab-pages",
		"iat": a.now().Unix(),
		"exp": a.now().Add(a.jwtExpiry).Unix(),
		// custom claims
		"domain": domain, // pass the domain so we can validate the signed domain matches the requested domain
		"code":   hex.EncodeToString(encryptedCode),
		"nonce":  base64.URLEncoding.EncodeToString(nonce),
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.jwtSigningKey)
}

// DecryptCode decodes the secureCode as a JWT token and validates its signature.
// It then decrypts the code from the token claims and returns it.
func (a *Auth) DecryptCode(jwt, domain string) (string, error) {
	claims, err := a.parseJWTClaims(jwt)
	if err != nil {
		return "", err
	}

	// get nonce and encryptedCode from the JWT claims
	encodedNonce, ok := claims["nonce"].(string)
	if !ok {
		return "", errInvalidNonce
	}

	nonce, err := base64.URLEncoding.DecodeString(encodedNonce)
	if err != nil {
		return "", errInvalidNonce
	}

	encryptedCode, ok := claims["code"].(string)
	if !ok {
		return "", errInvalidCode
	}

	cipherText, err := hex.DecodeString(encryptedCode)
	if err != nil {
		return "", err
	}

	aesGcm, err := a.newAesGcmCipher(domain, nonce)
	if err != nil {
		return "", err
	}

	decryptedCode, err := aesGcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(decryptedCode), nil
}

func (a *Auth) codeKey(domain string) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, []byte(a.authSecret), []byte(domain), []byte("PAGES_AUTH_CODE_ENCRYPTION_KEY"))

	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}

	return key, nil
}

func (a *Auth) parseJWTClaims(secureCode string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(secureCode, a.getSigningKey)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errInvalidToken
	}

	return claims, nil
}

func (a *Auth) getSigningKey(token *jwt.Token) (interface{}, error) {
	// Don't forget to validate the alg is what you expect:
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	return a.jwtSigningKey, nil
}

func (a *Auth) newAesGcmCipher(domain string, nonce []byte) (cipher.AEAD, error) {
	// get the same key for a domain
	key, err := a.codeKey(domain)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return nil, err
	}

	return aesGcm, nil
}
