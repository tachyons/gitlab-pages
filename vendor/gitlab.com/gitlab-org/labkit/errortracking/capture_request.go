package errortracking

import (
	"net/http"

	"github.com/getsentry/raven-go"
	"gitlab.com/gitlab-org/labkit/mask"
)

// WithRequest will capture details of the request along with the error
func WithRequest(r *http.Request) CaptureOption {
	return func(interfaces ravenInterfaces, _ raven.Extra) ravenInterfaces {
		cleanHeadersForRaven(r)
		return append(interfaces, raven.NewHttp(r))
	}
}

// cleanHeadersForRaven strips out information
// that shouldn't be send by the client
func cleanHeadersForRaven(r *http.Request) *raven.Http {
	if r == nil {
		return nil
	}

	report := raven.NewHttp(r)
	for header := range report.Headers {
		if mask.IsSensitiveHeader(header) {
			report.Headers[header] = mask.RedactionString
		}
	}

	params := r.URL.Query()
	for paramName := range params {
		if mask.IsSensitiveParam(paramName) {
			for i := range params[paramName] {
				params[paramName][i] = mask.RedactionString
			}
		}
	}
	report.Query = params.Encode()

	return report
}
