package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"
	"golang.org/x/crypto/hkdf"

	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

// nolint: gosec // auth constants, not credentials
// gosec: G101: Potential hardcoded credentials
const (
	apiURLUserTemplate     = "%s/api/v4/user"
	apiURLProjectTemplate  = "%s/api/v4/projects/%d/pages_access"
	authorizeURLTemplate   = "%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=%s"
	tokenURLTemplate       = "%s/oauth/token"
	callbackPath           = "/auth"
	authorizeProxyTemplate = "%s?domain=%s&state=%s"

	failAuthErrMsg             = "failed to authenticate request"
	fetchAccessTokenErrMsg     = "fetching access token failed"
	queryParameterErrMsg       = "failed to parse domain query parameter"
	saveSessionErrMsg          = "failed to save the session"
	domainQueryParameterErrMsg = "domain query parameter only supports http/https protocol"
)

var (
	errResponseNotOk     = errors.New("response was not ok")
	errAuthNotConfigured = errors.New("authentication is not configured")
	errGenerateKeys      = errors.New("could not generate auth keys")
)

// Auth handles authenticating users with GitLab API
type Auth struct {
	pagesDomain          string
	clientID             string
	clientSecret         string
	redirectURI          string
	internalGitlabServer string // used for exchanging OAuth code for token and Accessing API and checking if the user has access to the project
	publicGitlabServer   string // used for redirecting users to gitlab on the start of OAuth workflow
	authSecret           string
	authScope            string
	jwtSigningKey        []byte
	jwtExpiry            time.Duration
	apiClient            *http.Client
	store                sessions.Store
	now                  func() time.Time // allows to stub time.Now() easily in tests
	cookieSessionTimeout time.Duration
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}
type domain interface {
	GetProjectID(r *http.Request) uint64
	ServeNotFoundAuthFailed(w http.ResponseWriter, r *http.Request)
}

// TryAuthenticate tries to authenticate user and fetch access token if request is a callback to /auth?
func (a *Auth) TryAuthenticate(w http.ResponseWriter, r *http.Request, domains source.Source) bool {
	if a == nil {
		return false
	}

	session, err := a.checkSession(w, r)
	if err != nil {
		return true
	}

	// Request is for auth
	if r.URL.Path != callbackPath {
		return false
	}

	logRequest(r).Info("Receive OAuth authentication callback")

	if a.handleProxyingAuth(session, w, r, domains) {
		return true
	}

	// If callback is not successful
	errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		logRequest(r).WithField("error", errorParam).Warn("OAuth endpoint returned error")

		httperrors.Serve401(w)
		return true
	}

	if verifyCodeAndStateGiven(r) {
		a.checkAuthenticationResponse(session, w, r)
		return true
	}

	return false
}

func (a *Auth) checkAuthenticationResponse(session *hostSession, w http.ResponseWriter, r *http.Request) {
	if !validateState(r, session) {
		// State is NOT ok
		logRequest(r).Warn("Authentication state did not match expected")

		httperrors.Serve401(w)
		return
	}

	redirectURI, ok := session.Values["uri"].(string)
	if !ok {
		logRequest(r).Error("Can not extract redirect uri from session")
		httperrors.Serve500(w)
		return
	}

	decryptedCode, err := a.DecryptCode(r.URL.Query().Get("code"), getRequestDomain(r))
	if err != nil {
		logRequest(r).WithError(err).Error("failed to decrypt secure code")
		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve500(w)
		return
	}

	// Fetch access token with authorization code
	token, err := a.fetchAccessToken(r.Context(), decryptedCode)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			httperrors.Serve404(w)
			return
		}

		// Fetching token not OK
		logRequest(r).WithError(err).WithField(
			"redirect_uri", redirectURI,
		).Error(fetchAccessTokenErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r, errortracking.WithField("redirect_uri", redirectURI))

		httperrors.Serve503(w)
		return
	}

	// Store access token
	session.Values["access_token"] = token.AccessToken
	err = session.Save(r, w)
	if err != nil {
		logRequest(r).WithError(err).Error(saveSessionErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r)

		httperrors.Serve500(w)
		return
	}

	// Redirect back to requested URI
	logRequest(r).WithField(
		"redirect_uri", redirectURI,
	).Info("Authentication was successful, redirecting user back to requested page")

	http.Redirect(w, r, redirectURI, http.StatusFound)
}

func (a *Auth) domainAllowed(ctx context.Context, name string, domains source.Source) bool {
	isConfigured := (name == a.pagesDomain) || strings.HasSuffix(name, "."+a.pagesDomain)

	if isConfigured {
		return true
	}

	domain, err := domains.GetDomain(ctx, name)

	// domain exists and there is no error
	return (domain != nil && err == nil)
}

// nolint: gocyclo // TODO refactor this function https://gitlab.com/gitlab-org/gitlab-pages/-/issues/813
func (a *Auth) handleProxyingAuth(session *hostSession, w http.ResponseWriter, r *http.Request, domains source.Source) bool {
	// handle auth callback e.g. https://gitlab.io/auth?domain=domain&state=state
	if shouldProxyAuthToGitlab(r) {
		domain := r.URL.Query().Get("domain")
		state := r.URL.Query().Get("state")

		proxyurl, err := url.Parse(domain)
		if err != nil {
			logRequest(r).WithField("domain_query", domain).Error(queryParameterErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(err, r, errortracking.WithField("domain_query", domain))

			httperrors.Serve500(w)
			return true
		}

		// domain query param can only contain https or http URLs.
		if proxyurl.Scheme != "http" && proxyurl.Scheme != "https" {
			logRequest(r).WithField("domain_query", domain).Warn(domainQueryParameterErrMsg)

			httperrors.Serve401(w)
			return true
		}

		host, _, err := net.SplitHostPort(proxyurl.Host)
		if err != nil {
			host = proxyurl.Host
		}

		if !a.domainAllowed(r.Context(), host, domains) {
			logRequest(r).WithFields(logrus.Fields{
				"domain_query": domain,
				"domain_host":  host,
			}).Warn("Domain is not configured")
			httperrors.Serve401(w)
			return true
		}

		logRequest(r).WithField("domain_query", domain).Info("User is authenticating via domain")

		session.Values["proxy_auth_domain"] = domain

		err = session.Save(r, w)
		if err != nil {
			logRequest(r).WithError(err).Error(saveSessionErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(err, r)

			httperrors.Serve500(w)
			return true
		}

		url := fmt.Sprintf(authorizeURLTemplate, a.publicGitlabServer, a.clientID, a.redirectURI, state, a.authScope)

		logRequest(r).WithFields(logrus.Fields{
			"public_gitlab_server": a.publicGitlabServer,
			"domain_query":         domain,
		}).Info("Redirecting user to gitlab for oauth")

		http.Redirect(w, r, url, http.StatusFound)

		return true
	}

	// If auth request callback should be proxied to custom domain
	// redirect to originating domain set in the cookie as proxy_auth_domain
	if shouldProxyCallbackToCustomDomain(session) {
		// Get domain started auth process
		proxyDomain := session.Values["proxy_auth_domain"].(string)

		logRequest(r).WithField("proxy_auth_domain", proxyDomain).Info("Redirecting auth callback to custom domain")

		// Clear proxying from session
		delete(session.Values, "proxy_auth_domain")
		err := session.Save(r, w)
		if err != nil {
			logRequest(r).WithError(err).Error(saveSessionErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(err, r)

			httperrors.Serve500(w)
			return true
		}

		query := r.URL.Query()

		// prevent https://tools.ietf.org/html/rfc6749#section-10.6 and
		// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/262 by encrypting
		// and signing the OAuth code
		signedCode, err := a.EncryptAndSignCode(proxyDomain, query.Get("code"))
		if err != nil {
			logRequest(r).WithError(err).Error(saveSessionErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(err, r)

			httperrors.Serve503(w)
			return true
		}

		// prevent forwarding access token, more context on the security issue
		// https://gitlab.com/gitlab-org/gitlab/-/issues/285244#note_451266051
		query.Del("token")

		// replace code with signed code
		query.Set("code", signedCode)

		// Redirect pages to originating domain with code and state to finish
		// authentication process
		http.Redirect(w, r, proxyDomain+r.URL.Path+"?"+query.Encode(), http.StatusFound)
		return true
	}

	return false
}

func getRequestAddress(r *http.Request) string {
	if request.IsHTTPS(r) {
		return "https://" + r.Host + r.RequestURI
	}
	return "http://" + r.Host + r.RequestURI
}

func getRequestDomain(r *http.Request) string {
	if request.IsHTTPS(r) {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func shouldProxyAuthToGitlab(r *http.Request) bool {
	return r.URL.Query().Get("domain") != "" && r.URL.Query().Get("state") != ""
}

func shouldProxyCallbackToCustomDomain(session *hostSession) bool {
	return session.Values["proxy_auth_domain"] != nil
}

func validateState(r *http.Request, session *hostSession) bool {
	state := r.URL.Query().Get("state")
	if state == "" {
		// No state param
		return false
	}

	// Check state
	if session.Values["state"] == nil || session.Values["state"].(string) != state {
		// State does not match
		return false
	}

	// State ok
	return true
}

func verifyCodeAndStateGiven(r *http.Request) bool {
	return r.URL.Query().Get("code") != "" && r.URL.Query().Get("state") != ""
}

func (a *Auth) fetchAccessToken(ctx context.Context, code string) (tokenResponse, error) {
	token := tokenResponse{}

	// Prepare request
	fetchURL, err := url.Parse(fmt.Sprintf(tokenURLTemplate, a.internalGitlabServer))
	if err != nil {
		return token, err
	}

	content := url.Values{}
	content.Set("client_id", a.clientID)
	content.Set("client_secret", a.clientSecret)
	content.Set("code", code)
	content.Set("grant_type", "authorization_code")
	content.Set("redirect_uri", a.redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", fetchURL.String(), strings.NewReader(content.Encode()))
	if err != nil {
		return token, err
	}

	// Request token
	resp, err := a.apiClient.Do(req)
	if err != nil {
		return token, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errResponseNotOk
		errortracking.CaptureErrWithReqAndStackTrace(err, req)
		return token, err
	}

	// Parse response
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return token, err
	}

	return token, nil
}

func (a *Auth) checkSessionIsValid(w http.ResponseWriter, r *http.Request) *hostSession {
	session, err := a.checkSession(w, r)
	if err != nil {
		return nil
	}

	// redirect to /auth?domain=%s&state=%s
	if a.checkTokenExists(session, w, r) {
		return nil
	}

	return session
}

func (a *Auth) checkTokenExists(session *hostSession, w http.ResponseWriter, r *http.Request) bool {
	// If no access token redirect to OAuth login page
	if session.Values["access_token"] == nil {
		logRequest(r).Debug("No access token exists, redirecting user to OAuth2 login")

		// When the user tries to authenticate and reload the page concurrently,
		// gitlab pages might receive a authentication request with the state already set.
		// In these cases, we should re-use the state instead of creating a new one.
		if session.Values["state"] == nil {
			//Generate state hash and store requested address
			session.Values["state"] = base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(16))
		}

		session.Values["uri"] = getRequestAddress(r)

		// Clear possible proxying
		delete(session.Values, "proxy_auth_domain")

		err := session.Save(r, w)
		if err != nil {
			logRequest(r).WithError(err).Error(saveSessionErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(err, r)

			httperrors.Serve500(w)
			return true
		}

		// Because the pages domain might be in public suffix list, we have to
		// redirect to pages domain to trigger authorization flow
		http.Redirect(w, r, a.getProxyAddress(r, session.Values["state"].(string)), http.StatusFound)

		return true
	}
	return false
}

func (a *Auth) getProxyAddress(r *http.Request, state string) string {
	return fmt.Sprintf(authorizeProxyTemplate, a.redirectURI, getRequestDomain(r), state)
}

func destroySession(session *hostSession, w http.ResponseWriter, r *http.Request) {
	logRequest(r).Debug("Destroying session")

	// Invalidate access token and redirect back for refreshing and re-authenticating
	delete(session.Values, "access_token")
	err := session.Save(r, w)
	if err != nil {
		logRequest(r).WithError(err).Error(saveSessionErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r)

		httperrors.Serve500(w)
		return
	}

	http.Redirect(w, r, getRequestAddress(r), http.StatusFound)
}

// IsAuthSupported checks if pages is running with the authentication support
func (a *Auth) IsAuthSupported() bool {
	return a != nil
}

func (a *Auth) checkAuthentication(w http.ResponseWriter, r *http.Request, domain domain) bool {
	session := a.checkSessionIsValid(w, r)
	if session == nil {
		return true
	}

	projectID := domain.GetProjectID(r)
	// Access token exists, authorize request
	var url string
	if projectID > 0 {
		url = fmt.Sprintf(apiURLProjectTemplate, a.internalGitlabServer, projectID)
	} else {
		url = fmt.Sprintf(apiURLUserTemplate, a.internalGitlabServer)
	}
	req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
	if err != nil {
		logRequest(r).WithError(err).Error(failAuthErrMsg)
		errortracking.CaptureErrWithReqAndStackTrace(err, r)

		httperrors.Serve500(w)
		return true
	}

	req.Header.Add("Authorization", "Bearer "+session.Values["access_token"].(string))
	resp, err := a.apiClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			httperrors.Serve404(w)
			return true
		}

		logRequest(r).WithError(err).Error("Failed to retrieve info with token")
		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		// call serve404 handler when auth fails
		domain.ServeNotFoundAuthFailed(w, r)
		return true
	}

	defer resp.Body.Close()

	if checkResponseForInvalidToken(resp, session, w, r) {
		return true
	}

	if resp.StatusCode != http.StatusOK {
		// call serve404 handler when auth fails
		err := fmt.Errorf("unexpected response fetching access token status: %d", resp.StatusCode)
		logRequest(r).WithError(err).WithFields(log.Fields{
			"status":      resp.StatusCode,
			"status_text": resp.Status,
		}).Error("Unexpected response fetching access token")
		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		domain.ServeNotFoundAuthFailed(w, r)
		return true
	}

	return false
}

// CheckAuthenticationWithoutProject checks if user is authenticated and has a valid token
func (a *Auth) CheckAuthenticationWithoutProject(w http.ResponseWriter, r *http.Request, domain domain) bool {
	if a == nil {
		// No auth supported
		return false
	}

	return a.checkAuthentication(w, r, domain)
}

// GetTokenIfExists returns the token if it exists
func (a *Auth) GetTokenIfExists(w http.ResponseWriter, r *http.Request) (string, error) {
	if a == nil {
		return "", nil
	}

	session, err := a.checkSession(w, r)
	if err != nil {
		return "", errors.New("error retrieving the session")
	}

	if session.Values["access_token"] != nil {
		return session.Values["access_token"].(string), nil
	}

	return "", nil
}

// RequireAuth will trigger authentication flow if no token exists
func (a *Auth) RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	return a.checkSessionIsValid(w, r) == nil
}

// CheckAuthentication checks if user is authenticated and has access to the project
// will return contentServed = false when authFailed = true
func (a *Auth) CheckAuthentication(w http.ResponseWriter, r *http.Request, domain domain) bool {
	logRequest(r).Debug("Authenticate request")

	if a == nil {
		logRequest(r).Error(errAuthNotConfigured)
		errortracking.CaptureErrWithReqAndStackTrace(errAuthNotConfigured, r)

		httperrors.Serve500(w)
		return true
	}

	return a.checkAuthentication(w, r, domain)
}

// CheckResponseForInvalidToken checks response for invalid token and destroys session if it was invalid
func (a *Auth) CheckResponseForInvalidToken(w http.ResponseWriter, r *http.Request,
	resp *http.Response,
) bool {
	if a == nil {
		// No auth supported
		return false
	}

	session, err := a.checkSession(w, r)
	if err != nil {
		return true
	}

	if checkResponseForInvalidToken(resp, session, w, r) {
		return true
	}

	return false
}

func checkResponseForInvalidToken(resp *http.Response, session *hostSession, w http.ResponseWriter, r *http.Request) bool {
	if resp.StatusCode == http.StatusUnauthorized {
		errResp := errorResponse{}

		// Parse response
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		if err != nil {
			errortracking.CaptureErrWithReqAndStackTrace(err, r)
			return false
		}

		if errResp.Error == "invalid_token" {
			// Token is invalid
			logRequest(r).Warn("Access token was invalid, destroying session")

			destroySession(session, w, r)
			return true
		}
	}

	return false
}

func logRequest(r *http.Request) *logrus.Entry {
	return logging.LogRequest(r).WithField("state", r.URL.Query().Get("state"))
}

// generateKeys derives count hkdf keys from a secret, ensuring the key is
// the same for the same secret used across multiple instances
func generateKeys(secret string, count int) ([][]byte, error) {
	keys := make([][]byte, count)
	hkdfReader := hkdf.New(sha256.New, []byte(secret), []byte{}, []byte("PAGES_SIGNING_AND_ENCRYPTION_KEY"))

	for i := 0; i < count; i++ {
		key := make([]byte, 32)
		if _, err := io.ReadFull(hkdfReader, key); err != nil {
			return nil, err
		}

		keys[i] = key
	}

	if len(keys) < count {
		return nil, errGenerateKeys
	}

	return keys, nil
}

// Options carry required auth parameters used to populate Auth struct
type Options struct {
	PagesDomain          string
	StoreSecret          string
	ClientID             string
	ClientSecret         string
	RedirectURI          string
	InternalGitlabServer string
	PublicGitlabServer   string
	AuthScope            string
	AuthTimeout          time.Duration
	CookieSessionTimeout time.Duration
}

// New when authentication supported this will be used to create authentication handler
func New(options *Options) (*Auth, error) {
	// generate 3 keys, 2 for the cookie store and 1 for JWT signing
	keys, err := generateKeys(options.StoreSecret, 3)
	if err != nil {
		return nil, err
	}

	return &Auth{
		pagesDomain:          options.PagesDomain,
		clientID:             options.ClientID,
		clientSecret:         options.ClientSecret,
		redirectURI:          options.RedirectURI,
		internalGitlabServer: strings.TrimRight(options.InternalGitlabServer, "/"),
		publicGitlabServer:   strings.TrimRight(options.PublicGitlabServer, "/"),
		apiClient: &http.Client{
			Timeout:   options.AuthTimeout,
			Transport: httptransport.DefaultTransport,
		},
		store:                sessions.NewCookieStore(keys[0], keys[1]),
		authSecret:           options.StoreSecret,
		authScope:            options.AuthScope,
		jwtSigningKey:        keys[2],
		jwtExpiry:            time.Minute,
		now:                  time.Now,
		cookieSessionTimeout: options.CookieSessionTimeout,
	}, nil
}
