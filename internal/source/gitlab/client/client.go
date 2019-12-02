package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Client is a HTTP client to access Pages internal API
type Client struct {
	secretKey  []byte
	baseURL    *url.URL
	httpClient *http.Client
}

var (
	errUnknown      = errors.New("Unknown")
	errNoContent    = errors.New("No Content")
	errUnauthorized = errors.New("Unauthorized")
	errNotFound     = errors.New("Not Found")
)

// TODO make these values configurable https://gitlab.com/gitlab-org/gitlab-pages/issues/274
var tokenTimeout = 30 * time.Second
var connectionTimeout = 10 * time.Second

// NewClient initializes and returns new Client baseUrl is
// appConfig.GitLabServer secretKey is appConfig.GitLabAPISecretKey
func NewClient(baseURL string, secretKey []byte) *Client {
	url, err := url.Parse(baseURL)
	if err != nil {
		log.WithError(err).Fatal("could not parse GitLab server URL")
	}

	return &Client{
		secretKey: secretKey,
		baseURL:   url,
		httpClient: &http.Client{
			Timeout:   connectionTimeout,
			Transport: httptransport.Transport,
		},
	}
}

// NewFromConfig creates a new client from Config struct
func NewFromConfig(config Config) *Client {
	return NewClient(config.GitlabServerURL(), config.GitlabAPISecret())
}

// GetLookup returns a VirtualDomain configuration wrap into a Lookup for a
// given host
func (gc *Client) GetLookup(ctx context.Context, host string) api.Lookup {
	lookup := api.Lookup{Name: host}

	params := url.Values{}
	params.Set("host", host)

	resp, status, err := gc.get(ctx, "/api/v4/internal/pages", params)
	if resp != nil {
		defer resp.Body.Close()
	} else {
		err = errors.New("empty response returned")
	}

	lookup.Status = status
	lookup.Error = err

	if err != nil {
		return lookup
	}

	err = json.NewDecoder(resp.Body).Decode(&lookup.Domain)
	if err != nil {
		lookup.Error = err
		return lookup
	}

	return lookup
}

func (gc *Client) get(ctx context.Context, path string, params url.Values) (*http.Response, int, error) {
	endpoint, err := gc.endpoint(path, params)
	if err != nil {
		return nil, 0, err
	}

	req, err := gc.request(ctx, "GET", endpoint)
	if err != nil {
		return nil, 0, err
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return resp, resp.StatusCode, nil
	case resp.StatusCode == http.StatusNoContent:
		return resp, resp.StatusCode, errNoContent
	case resp.StatusCode == http.StatusUnauthorized:
		return resp, resp.StatusCode, errUnauthorized
	case resp.StatusCode == http.StatusNotFound:
		return resp, resp.StatusCode, errNotFound
	default:
		return resp, resp.StatusCode, errUnknown
	}
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
	req, err := http.NewRequest("GET", endpoint.String(), nil)
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
