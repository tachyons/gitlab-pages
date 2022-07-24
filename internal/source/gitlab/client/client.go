package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// ConnectionErrorMsg to be returned with `gc.Status` if Pages
// fails to connect to the internal GitLab API, times out
// or a 401 given that the credentials used are wrong
const ConnectionErrorMsg = "failed to connect to internal Pages API"

const transportClientName = "gitlab_internal_api"

// ErrUnauthorizedAPI is returned when resolving a domain with the GitLab API
// returns a http.StatusUnauthorized. This happens if the common secret file
// is not synced between gitlab-pages and gitlab-rails servers.
// See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/535 for more details.
var ErrUnauthorizedAPI = errors.New("pages endpoint unauthorized")

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
			Transport: httptransport.NewMeteredRoundTripper(
				correlation.NewInstrumentedRoundTripper(
					httptransport.DefaultTransport,
					correlation.WithClientName(transportClientName),
				),
				transportClientName,
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
func NewFromConfig(cfg *config.GitLab) (*Client, error) {
	return NewClient(cfg.InternalServer, cfg.APISecretKey, cfg.ClientHTTPTimeout, cfg.JWTTokenExpiration)
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
		metrics.DomainsSourceFailures.Inc()
		return api.Lookup{Name: host, Error: err}
	}

	if resp == nil {
		log.WithError(domain.ErrDomainDoesNotExist).WithFields(
			log.Fields{
				"correlation_id": correlation.ExtractFromContext(ctx),
				"lookup_name":    host,
			}).Error("unexpected nil response from gitlab")
		return api.Lookup{Name: host, Error: domain.ErrDomainDoesNotExist}
	}

	// ensure that entire response body has been read and close it, to make it
	// possible to reuse HTTP connection. In case of a JSON being invalid and
	// larger than 512 bytes, the response body will not be closed properly, thus
	// we need to close it manually in every case.
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	lookup := api.Lookup{Name: host}
	lookup.ParseDomain(resp.Body)

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

	// best effort to discard and close the response body
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// StatusNoContent means that a domain does not exist, it is not an error
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorizedAPI
	}

	return nil, fmt.Errorf("HTTP status: %d", resp.StatusCode)
}

func (gc *Client) endpoint(urlPath string, params url.Values) (*url.URL, error) {
	parsedPath, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	// fix for https://gitlab.com/gitlab-org/gitlab-pages/-/issues/587
	// ensure gc.baseURL.Path is still present and append new urlPath
	// it cleans double `/` in either path
	endpoint, err := gc.baseURL.Parse(path.Join(gc.baseURL.Path, parsedPath.Path))
	if err != nil {
		return nil, err
	}

	endpoint.RawQuery = params.Encode()

	return endpoint, nil
}

func (gc *Client) request(ctx context.Context, method string, endpoint *url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), nil)
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
	claims := jwt.RegisteredClaims{
		Issuer:    "gitlab-pages",
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(gc.jwtTokenExpiry)),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(gc.secretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}
