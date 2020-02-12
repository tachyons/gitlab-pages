package serverless

import "strings"

// Function represents a Knative service that is going to be invoked by the
// proxied request
type Function struct {
	Name      string // Name is a function name, it includes a "service name" component too
	Domain    string // Domain is a cluster base domain, used to route requests to apropriate service
	Namespace string // Namespace is a kubernetes namespace this function has been deployed to
}

// Host returns a function address that we are going to expose in the `Host:`
// header to make it possible to route a proxied request to appropriate service
// in a Knative cluster
func (f Function) Host() string {
	return strings.Join([]string{f.Name, f.Namespace, f.Domain}, ".")
}
