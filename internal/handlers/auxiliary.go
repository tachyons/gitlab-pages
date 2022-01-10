package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

func RedirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = request.SchemeHTTPS
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}
