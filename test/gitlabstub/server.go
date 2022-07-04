package gitlabstub

import (
	"net/http/httptest"
	"os"

	"github.com/gorilla/mux"
)

func NewUnstartedServer(opts ...Option) (*httptest.Server, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	conf := &config{
		pagesRoot: wd,
	}

	for _, so := range opts {
		so(conf)
	}

	if conf.pagesHandler == nil {
		conf.pagesHandler = defaultAPIHandler(conf.delay, conf.pagesRoot)
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/v4/internal/pages", conf.pagesHandler)

	authHandler := defaultAuthHandler()
	router.HandleFunc("/oauth/token", authHandler)

	userHandler := defaultUserHandler()
	router.HandleFunc("/api/v4/user", userHandler)

	router.HandleFunc("/api/v4/projects/{project_id:[0-9]+}/pages_access", handleAccessControlRequests)

	router.PathPrefix("/").HandlerFunc(handleAccessControlArtifactRequests)

	s := httptest.NewUnstartedServer(router)
	s.TLS = conf.tlsConfig

	return s, nil
}
