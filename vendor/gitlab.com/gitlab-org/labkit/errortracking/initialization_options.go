package errortracking

// The configuration for InjectCorrelationID
type initializationConfig struct {
	sentryDSN         string
	version           string
	sentryEnvironment string
	loggerName        string
}

// InitializationOption will configure a correlation handler
type InitializationOption func(*initializationConfig)

func applyInitializationOptions(opts []InitializationOption) initializationConfig {
	config := initializationConfig{}
	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithSentryDSN sets the sentry data source name
func WithSentryDSN(sentryDSN string) InitializationOption {
	return func(config *initializationConfig) {
		config.sentryDSN = sentryDSN
	}
}

// WithVersion is used to configure the version of the service
// that is currently running
func WithVersion(version string) InitializationOption {
	return func(config *initializationConfig) {
		config.version = version
	}
}

// WithSentryEnvironment sets the sentry environment
func WithSentryEnvironment(sentryEnvironment string) InitializationOption {
	return func(config *initializationConfig) {
		config.sentryEnvironment = sentryEnvironment
	}
}

// WithLoggerName sets the logger name
func WithLoggerName(loggerName string) InitializationOption {
	return func(config *initializationConfig) {
		config.loggerName = loggerName
	}
}
