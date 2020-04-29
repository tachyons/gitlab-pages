package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
)

const (
	defaultClientConnTimeout = 10 * time.Second
	defaultJWTTokenExpiry    = 30 * time.Second
)

func TestConnectionReuse(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/internal/pages", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		// we want to test for an invalid JSON that is larger than 512 bytes
		b := make([]byte, 513)
		for i := range b {
			b[i] = 'x'
		}

		w.Write(b)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := defaultClient(t, server.URL)
	reused := make(chan bool, 2)

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			reused <- connInfo.Reused
		},
	}

	ctx := httptrace.WithClientTrace(context.Background(), trace)
	client.GetLookup(ctx, "group.gitlab.io")
	client.GetLookup(ctx, "group.gitlab.io")

	require.False(t, <-reused)
	require.True(t, <-reused)
}

func TestNewValidBaseURL(t *testing.T) {
	_, err := NewClient("https://gitlab.com", secretKey(t), defaultClientConnTimeout, defaultJWTTokenExpiry)
	require.NoError(t, err)
}

func TestNewInvalidConfiguration(t *testing.T) {
	type args struct {
		baseURL           string
		secretKey         []byte
		connectionTimeout time.Duration
		jwtTokenExpiry    time.Duration
	}

	tests := []struct {
		name       string
		args       args
		wantErrMsg string
	}{
		{
			name: "invalid_api_url",
			args: args{
				baseURL:           "%",
				secretKey:         secretKey(t),
				connectionTimeout: defaultClientConnTimeout,
				jwtTokenExpiry:    defaultJWTTokenExpiry,
			},
			wantErrMsg: "invalid URL escape \"%\"",
		},
		{
			name: "invalid_api_url_empty",
			args: args{
				baseURL:           "",
				secretKey:         secretKey(t),
				connectionTimeout: defaultClientConnTimeout,
				jwtTokenExpiry:    defaultJWTTokenExpiry,
			},
			wantErrMsg: "GitLab API URL or API secret has not been provided",
		},
		{
			name: "invalid_api_secret_empty",
			args: args{
				baseURL:           "https://gitlab.com",
				secretKey:         []byte{},
				connectionTimeout: defaultClientConnTimeout,
				jwtTokenExpiry:    defaultJWTTokenExpiry,
			},
			wantErrMsg: "GitLab API URL or API secret has not been provided",
		},
		{
			name: "invalid_http_client_timeout",
			args: args{
				baseURL:           "https://gitlab.com",
				secretKey:         secretKey(t),
				connectionTimeout: 0,
				jwtTokenExpiry:    defaultJWTTokenExpiry,
			},
			wantErrMsg: "GitLab HTTP client connection timeout has not been provided",
		},
		{
			name: "invalid_jwt_token_expiry",
			args: args{
				baseURL:           "https://gitlab.com",
				secretKey:         secretKey(t),
				connectionTimeout: defaultClientConnTimeout,
				jwtTokenExpiry:    0,
			},
			wantErrMsg: "GitLab JWT token expiry has not been provided",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.baseURL, tt.args.secretKey, tt.args.connectionTimeout, tt.args.jwtTokenExpiry)
			require.Nil(t, got)
			require.NotNil(t, err)
			require.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}
func TestLookupForErrorResponses(t *testing.T) {
	tests := map[int]string{
		http.StatusUnauthorized: "HTTP status: 401",
		http.StatusNotFound:     "HTTP status: 404",
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

			client := defaultClient(t, server.URL)

			lookup := client.GetLookup(context.Background(), "group.gitlab.io")

			require.EqualError(t, lookup.Error, expectedError)
			require.Nil(t, lookup.Domain)
		})
	}
}

func TestMissingDomain(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/internal/pages", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := defaultClient(t, server.URL)

	lookup := client.GetLookup(context.Background(), "group.gitlab.io")

	require.NoError(t, lookup.Error)
	require.Nil(t, lookup.Domain)
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

	client := defaultClient(t, server.URL)

	lookup := client.GetLookup(context.Background(), "group.gitlab.io")
	require.NoError(t, lookup.Error)

	require.Equal(t, "foo", lookup.Domain.Certificate)
	require.Equal(t, "bar", lookup.Domain.Key)

	lookupPath := lookup.Domain.LookupPaths[0]
	require.Equal(t, 123, lookupPath.ProjectID)
	require.Equal(t, false, lookupPath.AccessControl)
	require.Equal(t, true, lookupPath.HTTPSOnly)
	require.Equal(t, "/myproject/", lookupPath.Prefix)

	require.Equal(t, "file", lookupPath.Source.Type)
	require.Equal(t, "mygroup/myproject/public/", lookupPath.Source.Path)
}

func validateToken(t *testing.T, tokenString string) {
	t.Helper()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return secretKey(t), nil
	})
	require.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.True(t, token.Valid)
	require.NotNil(t, claims["exp"])
	require.Equal(t, "gitlab-pages", claims["iss"])
}

func secretKey(t *testing.T) []byte {
	t.Helper()

	secretKey, err := base64.StdEncoding.DecodeString(fixture.GitLabAPISecretKey)
	require.NoError(t, err)

	return secretKey
}

func defaultClient(t *testing.T, url string) *Client {
	t.Helper()

	client, err := NewClient(url, secretKey(t), defaultClientConnTimeout, defaultJWTTokenExpiry)
	require.NoError(t, err)

	return client
}
