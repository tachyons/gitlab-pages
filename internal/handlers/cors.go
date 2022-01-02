package handlers

import (
	"net/http"

	"github.com/rs/cors"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{http.MethodGet, http.MethodHead}})
)

func CorsHandler(config *config.Config, handler http.Handler) http.Handler {
	if !config.General.DisableCrossOriginRequests {
		handler = corsHandler.Handler(handler)
	}
	return handler
}
