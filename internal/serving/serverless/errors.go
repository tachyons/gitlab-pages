package serverless

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

// NewErrorHandler returns a func(http.ResponseWriter, *http.Request, error)
// responsible for handling proxy errors
func NewErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// TODO provide serialized error message
		httperrors.Serve500(w)
	}
}
