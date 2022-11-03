package auth

import (
	"net/http"

	"github.com/gorilla/sessions"

	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

type hostSession struct {
	*sessions.Session
}

const sessionHostKey = "_session_host"

func (s *hostSession) Save(r *http.Request, w http.ResponseWriter) error {
	s.Session.Values[sessionHostKey] = r.Host

	return s.Session.Save(r, w)
}

func (a *Auth) getSessionFromStore(r *http.Request) (*hostSession, error) {
	session, err := a.store.Get(r, "gitlab-pages")

	if session != nil {
		// Cookie just for this domain
		session.Options.Path = "/"
		session.Options.HttpOnly = true
		session.Options.Secure = request.IsHTTPS(r)
		session.Options.MaxAge = int(a.cookieSessionTimeout.Seconds())

		if session.Values[sessionHostKey] == nil || session.Values[sessionHostKey] != r.Host {
			session.Values = make(map[interface{}]interface{})
		}
	}

	return &hostSession{session}, err
}

func (a *Auth) checkSession(w http.ResponseWriter, r *http.Request) (*hostSession, error) {
	// Create or get session
	session, errsession := a.getSessionFromStore(r)

	if errsession != nil {
		// Save cookie again
		errsave := session.Save(r, w)
		if errsave != nil {
			logRequest(r).WithError(errsave).Error(saveSessionErrMsg)
			errortracking.CaptureErrWithReqAndStackTrace(errsave, r)
			httperrors.Serve500(w)
			return nil, errsave
		}

		http.Redirect(w, r, getRequestAddress(r), http.StatusFound)
		return nil, errsession
	}

	return session, nil
}
