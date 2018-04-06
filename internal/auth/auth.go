package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

const (
	apiURLTemplate       = "%s/api/v4/projects/%d?access_token=%s"
	authorizeURLTemplate = "%s/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s"
	tokenURLTemplate     = "%s/oauth/token"
	tokenContentTemplate = "client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=%s"
	callbackPath         = "/auth"
)

// Auth handles authenticating users with GitLab API
type Auth struct {
	clientID     string
	clientSecret string
	redirectURI  string
	gitLabServer string
	store        *sessions.CookieStore
	apiClient    *http.Client
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

func (a *Auth) checkSession(w http.ResponseWriter, r *http.Request) bool {

	// Create or get session
	session, err := a.store.Get(r, "gitlab-pages")

	if err != nil {
		// Save cookie again
		session.Save(r, w)
		http.Redirect(w, r, getRequestAddress(r), 302)
		return true
	}

	return false
}

func (a *Auth) getSession(r *http.Request) *sessions.Session {
	session, _ := a.store.Get(r, "gitlab-pages")
	return session
}

// TryAuthenticate tries to authenticate user and fetch access token if request is a callback to auth
func (a *Auth) TryAuthenticate(w http.ResponseWriter, r *http.Request) bool {

	if a == nil {
		return false
	}

	if a.checkSession(w, r) {
		return true
	}

	session := a.getSession(r)

	// If callback from authentication and the state matches
	if r.URL.Path != callbackPath {
		return false
	}

	// If callback is not successful
	errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		httperrors.Serve401(w)
		return true
	}

	if verifyCodeAndStateGiven(r) {

		if !validateState(r, session) {
			// State is NOT ok
			httperrors.Serve401(w)
			return true
		}

		// Fetch access token with authorization code
		token, err := a.fetchAccessToken(r.URL.Query().Get("code"))

		// Fetching token not OK
		if err != nil {
			httperrors.Serve503(w)
			return true
		}

		// Store access token
		session.Values["access_token"] = token.AccessToken
		session.Save(r, w)

		// Redirect back to requested URI
		http.Redirect(w, r, session.Values["uri"].(string), 302)

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
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &token)
	if err != nil {
		return token, err
	}

	return token, nil
}

// CheckAuthentication checks if user is authenticated and has access to the project
func (a *Auth) CheckAuthentication(w http.ResponseWriter, r *http.Request, projectID int) bool {

	if a == nil {
		return false
	}

	if a.checkSession(w, r) {
		return true
	}

	session := a.getSession(r)

	// If no access token redirect to OAuth login page
	if session.Values["access_token"] == nil {

		// Generate state hash and store requested address
		state := base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(16))
		session.Values["state"] = state
		session.Values["uri"] = getRequestAddress(r)
		session.Save(r, w)

		// Redirect to OAuth login
		url := fmt.Sprintf(authorizeURLTemplate, a.gitLabServer, a.clientID, a.redirectURI, state)
		http.Redirect(w, r, url, 302)

		return true
	}

	// Access token exists, authorize request
	url := fmt.Sprintf(apiURLTemplate, a.gitLabServer, projectID, session.Values["access_token"].(string))
	resp, err := a.apiClient.Get(url)

	if checkResponseForInvalidToken(resp, err) {

		// Invalidate access token and redirect back for refreshing and re-authenticating
		delete(session.Values, "access_token")
		session.Save(r, w)

		http.Redirect(w, r, getRequestAddress(r), 302)

		return true
	}

	if err != nil || resp.StatusCode != 200 {
		httperrors.Serve401(w)
		return true
	}

	return false
}

func checkResponseForInvalidToken(resp *http.Response, err error) bool {
	if err == nil && resp.StatusCode == 401 {
		errResp := errorResponse{}

		// Parse response
		body, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		err = json.Unmarshal(body, &errResp)
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

	store := sessions.NewCookieStore([]byte(storeSecret))

	store.Options = &sessions.Options{
		Path:   "/",
		Domain: pagesDomain,
	}

	return &Auth{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		gitLabServer: strings.TrimRight(gitLabServer, "/"),
		store:        store,
		apiClient:    &http.Client{Timeout: 5 * time.Second},
	}
}
