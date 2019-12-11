package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Client is a HTTP client to access Pages internal API
type Client struct {
	secretKey  []byte
	baseURL    *url.URL
	httpClient *http.Client
}

// TODO make these values configurable https://gitlab.com/gitlab-org/gitlab-pages/issues/274
var tokenTimeout = 30 * time.Second
var connectionTimeout = 10 * time.Second

// NewClient initializes and returns new Client baseUrl is
// appConfig.GitLabServer secretKey is appConfig.GitLabAPISecretKey
func NewClient(baseURL string, secretKey []byte) (*Client, error) {
	if len(baseURL) == 0 || len(secretKey) == 0 {
		return nil, errors.New("GitLab API URL or API secret has not been provided")
	}

	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		secretKey: secretKey,
		baseURL:   url,
		httpClient: &http.Client{
			Timeout:   connectionTimeout,
			Transport: httptransport.Transport,
		},
	}, nil
}

// NewFromConfig creates a new client from Config struct
func NewFromConfig(config Config) (*Client, error) {
	return NewClient(config.GitlabServerURL(), config.GitlabAPISecret())
}

// GetLookup returns a VirtualDomain configuration wrap into a Lookup for a
// given host
func (gc *Client) GetLookup(ctx context.Context, host string) api.Lookup {
	params := url.Values{}
	params.Set("host", host)

	resp, err := gc.get(ctx, "/api/v4/internal/pages", params)
	if err != nil {
		return api.Lookup{Name: host, Error: err}
	}

	if resp == nil {
		return api.Lookup{Name: host}
	}

	lookup := api.Lookup{Name: host}
	lookup.Error = json.NewDecoder(resp.Body).Decode(&lookup.Domain)

	return lookup
}

func (gc *Client) get(ctx context.Context, path string, params url.Values) (*http.Response, error) {
	endpoint, err := gc.endpoint(path, params)
	if err != nil {
		return nil, err
	}

	req, err := gc.request(ctx, "GET", endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("unknown response")
	}

	// StatusOK means we should return the API response
	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	// StatusNoContent means that a domain does not exist, it is not an error
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	return nil, fmt.Errorf("HTTP status: %d", resp.StatusCode)
}

func (gc *Client) endpoint(path string, params url.Values) (*url.URL, error) {
	endpoint, err := gc.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	endpoint.RawQuery = params.Encode()

	return endpoint, nil
}

func (gc *Client) request(ctx context.Context, method string, endpoint *url.URL) (*http.Request, error) {
	req, err := http.NewRequest(method, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	token, err := gc.token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Gitlab-Pages-Api-Request", token)

	return req, nil
}

func (gc *Client) token() (string, error) {
	claims := jwt.StandardClaims{
		Issuer:    "gitlab-pages",
		ExpiresAt: time.Now().Add(tokenTimeout).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(gc.secretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}
