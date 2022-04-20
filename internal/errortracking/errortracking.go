package errortracking

import (
	"net/http"

	"gitlab.com/gitlab-org/labkit/errortracking"
)

// CaptureOption alias to avoid importing labkit/errortracking in internal packages
type CaptureOption = errortracking.CaptureOption

// WithField alias to avoid importing labkit/errortracking in internal packages
func WithField(key, value string) CaptureOption {
	return errortracking.WithField(key, value)
}

// CaptureErrWithReqAndStackTrace calls labkit's errortracking function and attaches the request, stack trace and any additional fields
func CaptureErrWithReqAndStackTrace(err error, r *http.Request, fields ...errortracking.CaptureOption) {
	opts := append(
		fields,
		errortracking.WithContext(r.Context()),
		errortracking.WithRequest(r),
		errortracking.WithStackTrace(),
	)

	errortracking.Capture(err, opts...)
}

// CaptureErrWithStackTrace calls labkit's errortracking function and attaches the stack trace and any additional fields
func CaptureErrWithStackTrace(err error, fields ...errortracking.CaptureOption) {
	opts := append(
		fields,
		errortracking.WithStackTrace(),
	)

	errortracking.Capture(err, opts...)
}
