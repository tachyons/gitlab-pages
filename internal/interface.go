package internal

import (
	"net/http"
)

// Artifact allows to handle artifact related requests
type Artifact interface {
	TryMakeRequest(w http.ResponseWriter, r *http.Request, token string, responseHandler func(*http.Response) bool) bool
}

// Auth handles the authentication logic
type Auth interface {
	IsAuthSupported() bool
	RequireAuth(w http.ResponseWriter, r *http.Request) bool
	GetTokenIfExists(w http.ResponseWriter, r *http.Request) (string, error)
	CheckResponseForInvalidToken(w http.ResponseWriter, r *http.Request, resp *http.Response) bool
}
