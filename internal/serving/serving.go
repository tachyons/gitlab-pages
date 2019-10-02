package serving

// Serving is an interface used to define a serving driver
type Serving interface {
	ServeFileHTTP(Handler) bool
	ServeNotFoundHTTP(Handler)
}
