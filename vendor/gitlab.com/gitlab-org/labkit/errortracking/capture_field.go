package errortracking

import (
	"github.com/getsentry/raven-go"
)

// WithField allows to add a custom field to the error
func WithField(key string, value string) CaptureOption {
	return func(interfaces ravenInterfaces, extra raven.Extra) ravenInterfaces {
		extra[key] = value
		return interfaces
	}
}
