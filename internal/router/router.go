package router

import "net/http"

type middleware = func(http.Handler) http.Handler

type server struct {
	handler            *http.ServeMux
	defaultMiddlewares []middleware
}

func NewRouter(middlewares ...middleware) server {
	return server{
		handler:            http.NewServeMux(),
		defaultMiddlewares: middlewares,
	}
}

func (s server) Handle(route string, handler http.Handler, middlewares ...middleware) {
	ms := append(s.defaultMiddlewares, middlewares...)

	for i := len(ms) - 1; i >= 0; i-- {
		handler = ms[i](handler)
	}

	s.handler.Handle(route, handler)
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}
