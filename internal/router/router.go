package router

import "net/http"

// middleware alias to avoid the long function description internally
type middleware = func(http.Handler) http.Handler

type Router struct {
	*http.ServeMux
	defaultMiddlewares []middleware
}

// NewRouter creates a new Server. The given middlewares are be executed in the given order.
func NewRouter(middlewares ...middleware) Router {
	return Router{
		ServeMux:           http.NewServeMux(),
		defaultMiddlewares: middlewares,
	}
}

// Handle registers a new handler for the given pattern. The optional middlewares are executed in
// the given order, wrapping the given handler.
func (s Router) Handle(route string, handler http.Handler, middlewares ...middleware) {
	ms := append(s.defaultMiddlewares, middlewares...)

	for i := len(ms) - 1; i >= 0; i-- {
		handler = ms[i](handler)
	}

	s.ServeMux.Handle(route, handler)
}
