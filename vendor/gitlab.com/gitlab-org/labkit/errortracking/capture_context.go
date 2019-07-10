package errortracking

import (
	"context"

	"github.com/getsentry/raven-go"
	"gitlab.com/gitlab-org/labkit/correlation"
)

const ravenSentryExtraKey = "gitlab.CorrelationID"

// WithContext will extract information from the context to add to the error
func WithContext(ctx context.Context) CaptureOption {
	return func(interfaces ravenInterfaces, extra raven.Extra) ravenInterfaces {

		correlationID := correlation.ExtractFromContext(ctx)
		if correlationID != "" {
			extra[ravenSentryExtraKey] = correlationID
		}

		return interfaces
	}
}
