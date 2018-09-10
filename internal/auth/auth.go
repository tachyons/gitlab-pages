package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
)

const (
	apiURLUserTemplate     = "%s/api/v4/user"
	apiURLProjectTemplate  = "%s/api/v4/projects/%d/pages_access"
	authorizeURLTemplate   = "%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s"
	tokenURLTemplate       = "%s/oauth/token"
	tokenContentTemplate   = "client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s"
	callbackPath           = "/auth"
	authorizeProxyTemplate = "%s?domain=%s&state=%s"
)

// Auth handles authenticating users with GitLab API
type Auth struct {
	pagesDomain  string
	clientID     string
	clientSecret string
	redirectURI  string
	gitLabServer string
	apiClient    *http.Client
	store        sessions.Store
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

func (a *Auth) getSessionFromStore(r *http.Request) (*sessions.Session, error) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	session, err := a.store.Get(r, "gitlab-pages")

	if session != nil {
		// Cookie just for this domain
		session.Options = &sessions.Options{
			Path:   "/",
			Domain: host,
		}
	}

	return session, err
}

func (a *Auth) checkSession(w http.ResponseWriter, r *http.Request) (*sessions.Session, error) {

	// Create or get session
	session, errsession := a.getSessionFromStore(r)

	if errsession != nil {
		// Save cookie again
		errsave := session.Save(r, w)
		if errsave != nil {
			log.WithError(errsave).Error("Failed to save the session")
			httperrors.Serve500(w)
			return nil, errsave
		}

		http.Redirect(w, r, getRequestAddress(r), 302)
		return nil, errsession
	}

	return session, nil
}

// TryAuthenticate tries to authenticate user and fetch access token if request is a callback to auth
func (a *Auth) TryAuthenticate(w http.ResponseWriter, r *http.Request, dm domain.Map, lock *sync.RWMutex) bool {

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

	log.Debug("Authentication callback")

	if a.handleProxyingAuth(session, w, r, dm, lock) {
		return true
	}

	// If callback is not successful
	errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		log.WithField("error", errorParam).Debug("OAuth endpoint returned error")

		httperrors.Serve401(w)
		return true
	}

	if verifyCodeAndStateGiven(r) {
		a.checkAuthenticationResponse(session, w, r)
		return true
	}

	return false
}

func (a *Auth) checkAuthenticationResponse(session *sessions.Session, w http.ResponseWriter, r *http.Request) {

	if !validateState(r, session) {
		// State is NOT ok
		log.Debug("Authentication state did not match expected")

		httperrors.Serve401(w)
		return
	}

	// Fetch access token with authorization code
	token, err := a.fetchAccessToken(r.URL.Query().Get("code"))

	// Fetching token not OK
	if err != nil {
		log.WithError(err).Debug("Fetching access token failed")

		httperrors.Serve503(w)
		return
	}

	// Store access token
	session.Values["access_token"] = token.AccessToken
	err = session.Save(r, w)
	if err != nil {
		log.WithError(err).Error("Failed to save the session")
		httperrors.Serve500(w)
		return
	}

	// Redirect back to requested URI
	log.Debug("Authentication was successful, redirecting user back to requested page")

	http.Redirect(w, r, session.Values["uri"].(string), 302)
}

func (a *Auth) domainAllowed(domain string, dm domain.Map, lock *sync.RWMutex) bool {
	lock.RLock()
	defer lock.RUnlock()
	_, present := dm[domain]
	return strings.HasSuffix(strings.ToLower(domain), a.pagesDomain) || present
}

func (a *Auth) handleProxyingAuth(session *sessions.Session, w http.ResponseWriter, r *http.Request, dm domain.Map, lock *sync.RWMutex) bool {
	// If request is for authenticating via custom domain
	if shouldProxyAuth(r) {
		domain := r.URL.Query().Get("domain")
		state := r.URL.Query().Get("state")

		proxyurl, err := url.Parse(domain)
		if err != nil {
			log.WithField("domain", domain).Error("Failed to parse domain query parameter")
			httperrors.Serve500(w)
			return true
		}
		host, _, err := net.SplitHostPort(proxyurl.Host)
		if err != nil {
			host = proxyurl.Host
		}

		if !a.domainAllowed(host, dm, lock) {
			log.WithField("domain", host).Debug("Domain is not configured")
			httperrors.Serve401(w)
			return true
		}

		log.WithField("domain", domain).Debug("User is authenticating via domain")

		session.Values["proxy_auth_domain"] = domain

		err = session.Save(r, w)
		if err != nil {
			log.WithError(err).Error("Failed to save the session")
			httperrors.Serve500(w)
			return true
		}

		url := fmt.Sprintf(authorizeURLTemplate, a.gitLabServer, a.clientID, a.redirectURI, state)
		http.Redirect(w, r, url, 302)

		return true
	}

	// If auth request callback should be proxied to custom domain
	if shouldProxyCallbackToCustomDomain(r, session) {
		// Auth request is from custom domain, proxy callback there
		log.Debug("Redirecting auth callback to custom domain")

		// Store access token
		proxyDomain := session.Values["proxy_auth_domain"].(string)

		// Clear proxying from session
		delete(session.Values, "proxy_auth_domain")
		err := session.Save(r, w)
		if err != nil {
			log.WithError(err).Error("Failed to save the session")
			httperrors.Serve500(w)
			return true
		}

		// Redirect pages under custom domain
		http.Redirect(w, r, proxyDomain+r.URL.Path+"?"+r.URL.RawQuery, 302)

		return true
	}

	return false
}

func getRequestAddress(r *http.Request) string {
	if r.TLS != nil {
		return "https://" + r.Host + r.RequestURI
	}
	return "http://" + r.Host + r.RequestURI
}

func getRequestDomain(r *http.Request) string {
	if r.TLS != nil {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func shouldProxyAuth(r *http.Request) bool {
	return r.URL.Query().Get("domain") != "" && r.URL.Query().Get("state") != ""
}

func shouldProxyCallbackToCustomDomain(r *http.Request, session *sessions.Session) bool {
	return session.Values["proxy_auth_domain"] != nil
}

func validateState(r *http.Request, session *sessions.Session) bool {
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

func (a *Auth) fetchAccessToken(code string) (tokenResponse, error) {
	token := tokenResponse{}

	// Prepare request
	url := fmt.Sprintf(tokenURLTemplate, a.gitLabServer)
	content := fmt.Sprintf(tokenContentTemplate, a.clientID, a.clientSecret, code, a.redirectURI)
	req, err := http.NewRequest("POST", url, strings.NewReader(content))

	if err != nil {
		return token, err
	}

	// Request token
	resp, err := a.apiClient.Do(req)

	if err != nil {
		return token, err
	}

	if resp.StatusCode != 200 {
		return token, errors.New("response was not OK")
	}

	// Parse response
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return token, err
	}

	return token, nil
}

func (a *Auth) checkTokenExists(session *sessions.Session, w http.ResponseWriter, r *http.Request) bool {
	// If no access token redirect to OAuth login page
	if session.Values["access_token"] == nil {
		log.Debug("No access token exists, redirecting user to OAuth2 login")

		// Generate state hash and store requested address
		state := base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(16))
		session.Values["state"] = state
		session.Values["uri"] = getRequestAddress(r)

		// Clear possible proxying
		delete(session.Values, "proxy_auth_domain")

		err := session.Save(r, w)
		if err != nil {
			log.WithError(err).Error("Failed to save the session")
			httperrors.Serve500(w)
			return true
		}

		// Because the pages domain might be in public suffix list, we have to
		// redirect to pages domain to trigger authorization flow
		http.Redirect(w, r, a.getProxyAddress(r, state), 302)

		return true
	}
	return false
}

func (a *Auth) getProxyAddress(r *http.Request, state string) string {
	return fmt.Sprintf(authorizeProxyTemplate, a.redirectURI, getRequestDomain(r), state)
}

func destroySession(session *sessions.Session, w http.ResponseWriter, r *http.Request) {
	log.Debug("Destroying session")

	// Invalidate access token and redirect back for refreshing and re-authenticating
	delete(session.Values, "access_token")
	err := session.Save(r, w)
	if err != nil {
		log.WithError(err).Error("Failed to save the session")
		httperrors.Serve500(w)
		return
	}

	http.Redirect(w, r, getRequestAddress(r), 302)
}

// IsAuthSupported checks if pages is running with the authentication support
func (a *Auth) IsAuthSupported() bool {
	if a == nil {
		return false
	}
	return true
}

// CheckAuthenticationWithoutProject checks if user is authenticated and has a valid token
func (a *Auth) CheckAuthenticationWithoutProject(w http.ResponseWriter, r *http.Request) bool {

	if a == nil {
		// No auth supported
		return false
	}

	session, err := a.checkSession(w, r)
	if err != nil {
		return true
	}

	if a.checkTokenExists(session, w, r) {
		return true
	}

	// Access token exists, authorize request
	url := fmt.Sprintf(apiURLUserTemplate, a.gitLabServer)
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.WithError(err).Debug("Failed to authenticate request")

		httperrors.Serve500(w)
		return true
	}

	req.Header.Add("Authorization", "Bearer "+session.Values["access_token"].(string))
	resp, err := a.apiClient.Do(req)

	if checkResponseForInvalidToken(resp, err) {
		log.Debug("Access token was invalid, destroying session")

		destroySession(session, w, r)
		return true
	}

	if err != nil || resp.StatusCode != 200 {
		// We return 404 if for some reason token is not valid to avoid (not) existence leak
		if err != nil {
			log.WithError(err).Debug("Failed to retrieve info with token")
		}

		httperrors.Serve404(w)
		return true
	}

	return false
}

// CheckAuthentication checks if user is authenticated and has access to the project
func (a *Auth) CheckAuthentication(w http.ResponseWriter, r *http.Request, projectID uint64) bool {

	if a == nil {
		log.Debug("Authentication is not configured")
		httperrors.Serve500(w)
		return true
	}

	session, err := a.checkSession(w, r)
	if err != nil {
		return true
	}

	if a.checkTokenExists(session, w, r) {
		return true
	}

	// Access token exists, authorize request
	url := fmt.Sprintf(apiURLProjectTemplate, a.gitLabServer, projectID)
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		httperrors.Serve500(w)
		return true
	}

	req.Header.Add("Authorization", "Bearer "+session.Values["access_token"].(string))
	resp, err := a.apiClient.Do(req)

	if checkResponseForInvalidToken(resp, err) {
		log.Debug("Access token was invalid, destroying session")

		destroySession(session, w, r)
		return true
	}

	if err != nil || resp.StatusCode != 200 {
		if err != nil {
			log.WithError(err).Debug("Failed to retrieve info with token")
		}

		// We return 404 if user has no access to avoid user knowing if the pages really existed or not
		httperrors.Serve404(w)
		return true
	}

	return false
}

func checkResponseForInvalidToken(resp *http.Response, err error) bool {
	if err == nil && resp.StatusCode == 401 {
		errResp := errorResponse{}

		// Parse response
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		if err != nil {
			return false
		}

		if errResp.Error == "invalid_token" {
			// Token is invalid
			return true
		}
	}

	return false
}

// New when authentication supported this will be used to create authentication handler
func New(pagesDomain string, storeSecret string, clientID string, clientSecret string,
	redirectURI string, gitLabServer string) *Auth {
	return &Auth{
		pagesDomain:  pagesDomain,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		gitLabServer: strings.TrimRight(gitLabServer, "/"),
		apiClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: httptransport.Transport,
		},
		store: sessions.NewCookieStore([]byte(storeSecret)),
	}
}
