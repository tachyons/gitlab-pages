package main

import (
	"gitlab.com/gitlab-org/labkit/errortracking"
)

func capturingFatal(err error, fields ...errortracking.CaptureOption) {
	fields = append(fields, errortracking.WithStackTrace())
	errortracking.Capture(err, fields...)
	fatal(err, "capturing fatal")
}
