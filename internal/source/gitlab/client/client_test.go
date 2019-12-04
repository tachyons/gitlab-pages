package client

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	encodedSecret = "e41rcFh7XBA7sNABWVCe2AZvxMsy6QDtJ8S9Ql1UiN8=" // 32 bytes, base64 encoded
)

func TestNewValidBaseURL(t *testing.T) {
	_, err := NewClient("https://gitlab.com", secretKey())
	require.NoError(t, err)
}

func TestNewInvalidBaseURL(t *testing.T) {
	t.Run("when API URL is not valid", func(t *testing.T) {
		client, err := NewClient("%", secretKey())

		require.Error(t, err)
		require.Nil(t, client)
	})

	t.Run("when API URL is empty", func(t *testing.T) {
		client, err := NewClient("", secretKey())

		require.Nil(t, client)
		require.EqualError(t, err, "GitLab API URL or API secret has not been provided")
	})

	t.Run("when API secret is empty", func(t *testing.T) {
		client, err := NewClient("https://gitlab.com", []byte{})

		require.Nil(t, client)
		require.EqualError(t, err, "GitLab API URL or API secret has not been provided")
	})
}

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

			client, err := NewClient(server.URL, secretKey())
			require.NoError(t, err)

			actual, err := client.GetVirtualDomain("group.gitlab.io")

			require.EqualError(t, err, expectedError)
			require.Nil(t, actual)
		})
	}
}

func TestGetVirtualDomainAuthenticatedRequest(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/internal/pages", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "group.gitlab.io", r.FormValue("host"))

		validateToken(t, r.Header.Get("Gitlab-Pages-Api-Request"))

		response := `{
			"certificate": "foo",
			"key": "bar",
			"lookup_paths": [
				{
					"project_id": 123,
					"access_control": false,
					"source": {
						"type": "file",
						"path": "mygroup/myproject/public/"
					},
					"https_only": true,
					"prefix": "/myproject/"
				}
			]
		}`

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := NewClient(server.URL, secretKey())
	require.NoError(t, err)

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

func validateToken(t *testing.T, tokenString string) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return secretKey(), nil
	})
	require.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.True(t, token.Valid)
	require.NotNil(t, claims["exp"])
	require.Equal(t, "gitlab-pages", claims["iss"])
}

func secretKey() []byte {
	secretKey, _ := base64.StdEncoding.DecodeString(encodedSecret)
	return secretKey
}
