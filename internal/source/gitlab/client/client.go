package client

import (
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

// GetVirtualDomain returns VirtualDomain configuration for the given host. It
// returns an error if non-nil `*api.VirtualDomain` can not be retuned.
func (gc *Client) GetVirtualDomain(host string) (*api.VirtualDomain, error) {
	params := url.Values{}
	params.Set("host", host)

	resp, err := gc.get("/api/v4/internal/pages", params)
	if resp != nil {
		defer resp.Body.Close()
	} else {
		return nil, errors.New("empty response returned")
	}

	if err != nil {
		return nil, err
	}

	var domain api.VirtualDomain
	err = json.NewDecoder(resp.Body).Decode(&domain)
	if err != nil {
		return nil, err
	}

	return &domain, nil
}

func (gc *Client) get(path string, params url.Values) (*http.Response, error) {
	endpoint, err := gc.endpoint(path, params)
	if err != nil {
		return nil, err
	}

	req, err := gc.request("GET", endpoint)
	if err != nil {
		return nil, err
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return resp, nil
	case resp.StatusCode == http.StatusNoContent:
		return resp, errNoContent
	case resp.StatusCode == http.StatusUnauthorized:
		return resp, errUnauthorized
	case resp.StatusCode == http.StatusNotFound:
		return resp, errNotFound
	default:
		return resp, errUnknown
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

func (gc *Client) request(method string, endpoint *url.URL) (*http.Request, error) {
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

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
