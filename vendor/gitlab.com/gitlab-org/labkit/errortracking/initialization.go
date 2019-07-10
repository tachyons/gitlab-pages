package errortracking

import (
	"github.com/getsentry/raven-go"
)

// Initialize will initialize error reporting
func Initialize(opts ...InitializationOption) error {
	config := applyInitializationOptions(opts)

	if err := raven.SetDSN(config.sentryDSN); err != nil {
		return err
	}

	raven.SetEnvironment(config.sentryEnvironment)

	if config.version != "" {
		raven.SetRelease(config.version)
	}

	if config.loggerName != "" {
		raven.SetDefaultLoggerName(config.loggerName)
	}

	return nil
}
