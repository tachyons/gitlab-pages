package errortracking

import (
	"reflect"

	"github.com/getsentry/raven-go"
)

// Capture will report an error to the error reporting service
func Capture(err error, opts ...CaptureOption) {
	var interfaces []raven.Interface
	extra := raven.Extra{}

	for _, v := range opts {
		interfaces = v(interfaces, extra)
	}

	client := raven.DefaultClient

	exception := &raven.Exception{
		Stacktrace: raven.NewStacktrace(2, 3, nil),
		Value:      err.Error(),
		Type:       reflect.TypeOf(err).String(),
	}
	interfaces = append(interfaces, exception)

	packet := raven.NewPacketWithExtra(err.Error(), extra, interfaces...)
	client.Capture(packet, nil)
}
