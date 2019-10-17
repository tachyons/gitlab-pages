package gitlab

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
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

// NewClient initializes and returns new Client
func NewClient(baseURL string, secretKey []byte) (*Client, error) {
	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		secretKey: secretKey,
		baseURL:   url,
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: httptransport.Transport,
		},
	}, nil
}

// GetVirtualDomain returns VirtualDomain configuration for the given host
func (gc *Client) GetVirtualDomain(host string) (*VirtualDomain, error) {
	params := map[string]string{"host": host}

	resp, err := gc.get("/api/v4/internal/pages", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var domain VirtualDomain
	err = json.NewDecoder(resp.Body).Decode(&domain)
	if err != nil {
		return nil, err
	}

	return &domain, nil
}

func (gc *Client) get(path string, params map[string]string) (*http.Response, error) {
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

func (gc *Client) endpoint(path string, params map[string]string) (*url.URL, error) {
	endpoint, err := gc.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}
	endpoint.RawQuery = values.Encode()

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
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(gc.secretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}
