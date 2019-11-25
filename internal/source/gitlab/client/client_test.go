package client

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	encodedSecret = "e41rcFh7XBA7sNABWVCe2AZvxMsy6QDtJ8S9Ql1UiN8=" // 32 bytes, base64 encoded
)

func TestGetVirtualDomainForErrorResponses(t *testing.T) {
	tests := map[int]string{
		http.StatusNoContent:    "No Content",
		http.StatusUnauthorized: "Unauthorized",
		http.StatusNotFound:     "Not Found",
	}

	for statusCode, expectedError := range tests {
		name := fmt.Sprintf("%d %s", statusCode, expectedError)
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()

			mux.HandleFunc("/api/v4/internal/pages", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			client := NewClient(server.URL, secretKey())

			actual, err := client.GetVirtualDomain("group.gitlab.io")

			require.EqualError(t, err, expectedError)
			require.Nil(t, actual)
		})
	}
}

func TestGetVirtualDomainAuthenticatedRequest(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/internal/pages", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "group.gitlab.io", r.FormValue("host"))

		if checkRequest(r.Header.Get("Gitlab-Pages-Api-Request")) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"certificate":"foo","key":"bar","lookup_paths":[{"project_id":123,"access_control":false,"source":{"type":"file","path":"mygroup/myproject/public/"},"https_only":true,"prefix":"/myproject/"}]}`)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL, secretKey())

	actual, err := client.GetVirtualDomain("group.gitlab.io")
	require.NoError(t, err)

	require.Equal(t, "foo", actual.Certificate)
	require.Equal(t, "bar", actual.Key)

	lookupPath := actual.LookupPaths[0]
	require.Equal(t, 123, lookupPath.ProjectID)
	require.Equal(t, false, lookupPath.AccessControl)
	require.Equal(t, true, lookupPath.HTTPSOnly)
	require.Equal(t, "/myproject/", lookupPath.Prefix)

	require.Equal(t, "file", lookupPath.Source.Type)
	require.Equal(t, "mygroup/myproject/public/", lookupPath.Source.Path)
}

func checkRequest(tokenString string) bool {
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return secretKey(), nil
	})

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return false
	}

	if _, ok := claims["exp"]; !ok {
		return false
	}

	return claims["iss"] == "gitlab-pages"
}

func secretKey() []byte {
	secretKey, _ := base64.StdEncoding.DecodeString(encodedSecret)
	return secretKey
}
