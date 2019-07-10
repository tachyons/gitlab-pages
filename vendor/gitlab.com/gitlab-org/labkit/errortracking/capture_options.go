package errortracking

import (
	"github.com/getsentry/raven-go"
)

type ravenInterfaces []raven.Interface

// CaptureOption will configure how an error is captured
type CaptureOption func(ravenInterfaces, raven.Extra) ravenInterfaces
