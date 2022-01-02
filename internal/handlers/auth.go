package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func Authorization(auth *auth.Auth, handler http.Handler) http.Handler {
	return auth.AuthorizationMiddleware(handler)
}

func Authentication(auth *auth.Auth, s source.Source, handler http.Handler) http.Handler {
	return auth.AuthenticationMiddleware(handler, s)
}
