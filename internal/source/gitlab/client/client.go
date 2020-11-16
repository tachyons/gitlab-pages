package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// ConnectionErrorMsg to be returned with `gc.Status` if Pages
// fails to connect to the internal GitLab API, times out
// or a 401 given that the credentials used are wrong
const ConnectionErrorMsg = "failed to connect to internal Pages API"

// ErrDomainDoesNotExist should be returned when we get a 204 from the API when
// GetLookup is called
var ErrDomainDoesNotExist = errors.New("domain does not exist")

// Client is a HTTP client to access Pages internal API
type Client struct {
	secretKey      []byte
	baseURL        *url.URL
	httpClient     *http.Client
	jwtTokenExpiry time.Duration
}

// NewClient initializes and returns new Client baseUrl is
// appConfig.InternalGitLabServer secretKey is appConfig.GitLabAPISecretKey
func NewClient(baseURL string, secretKey []byte, connectionTimeout, jwtTokenExpiry time.Duration) (*Client, error) {
	if len(baseURL) == 0 || len(secretKey) == 0 {
		return nil, errors.New("GitLab API URL or API secret has not been provided")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	if connectionTimeout == 0 {
		return nil, errors.New("GitLab HTTP client connection timeout has not been provided")
	}

	if jwtTokenExpiry == 0 {
		return nil, errors.New("GitLab JWT token expiry has not been provided")
	}

	return &Client{
		secretKey: secretKey,
		baseURL:   parsedURL,
		httpClient: &http.Client{
			Timeout: connectionTimeout,
			Transport: httptransport.NewTransportWithMetrics(
				"gitlab_internal_api",
				metrics.DomainsSourceAPITraceDuration,
				metrics.DomainsSourceAPICallDuration,
				metrics.DomainsSourceAPIReqTotal,
				httptransport.DefaultTTFBTimeout,
			),
		},
		jwtTokenExpiry: jwtTokenExpiry,
	}, nil
}

// NewFromConfig creates a new client from Config struct
func NewFromConfig(config Config) (*Client, error) {
	return NewClient(config.InternalGitLabServerURL(), config.GitlabAPISecret(), config.GitlabClientConnectionTimeout(), config.GitlabJWTTokenExpiry())
}

// Resolve returns a VirtualDomain configuration wrapped into a Lookup for a
// given host. It implements api.Resolve type.
func (gc *Client) Resolve(ctx context.Context, host string) *api.Lookup {
	lookup := gc.GetLookup(ctx, host)

	return &lookup
}

// GetLookup returns a VirtualDomain configuration wrapped into a Lookup for a
// given host
func (gc *Client) GetLookup(ctx context.Context, host string) api.Lookup {
	params := url.Values{}
	params.Set("host", host)

	resp, err := gc.get(ctx, "/api/v4/internal/pages", params)
	if err != nil {
		return api.Lookup{Name: host, Error: err}
	}

	dres, err := httputil.DumpResponse(resp, true)
	fmt.Printf("dres: %s\n%+v\n", dres, err)
	// ensure that entire response body has been read and close it, to make it
	// possible to reuse HTTP connection. In case of a JSON being invalid and
	// larger than 512 bytes, the response body will not be closed properly, thus
	// we need to close it manually in every case.
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	lookup := api.Lookup{Name: host}
	if resp.StatusCode == http.StatusNoContent {
		lookup.Error = ErrDomainDoesNotExist
	}

	lookup.Error = json.NewDecoder(resp.Body).Decode(&lookup.Domain)

	return lookup
}

// Status checks that Pages can reach the rails internal Pages API
// for source domain configuration.
// Timeout is the same as -gitlab-client-http-timeout
func (gc *Client) Status() error {
	res, err := gc.get(context.Background(), "/api/v4/internal/pages/status", url.Values{})
	if err != nil {
		return fmt.Errorf("%s: %v", ConnectionErrorMsg, err)
	}

	if res != nil && res.Body != nil {
		res.Body.Close()
	}

	return nil
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
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return resp, nil
	}

	var apiErr *Err
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return nil, err
	}

	// return nil, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	return nil, apiErr
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

	dreq, err := httputil.DumpRequestOut(req, true)
	fmt.Printf("dreq: %s\n%+v\n", dreq, err)
	return req, nil
}

func (gc *Client) token() (string, error) {
	claims := jwt.StandardClaims{
		Issuer:    "gitlab-pages",
		ExpiresAt: time.Now().UTC().Add(gc.jwtTokenExpiry).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(gc.secretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Err describes any error response received from the API
type Err struct {
	Status  int
	Message string `json:"message"`
	Err     string `json:"error"`
}

func (e *Err) Error() string {
	if e.Err != "" {
		return fmt.Sprintf("status: %d error: %s", e.Status, e.Err)
	}

	return fmt.Sprintf("status: %d error: %s", e.Status, e.Message)
}
