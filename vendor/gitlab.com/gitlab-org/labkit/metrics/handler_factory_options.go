package metrics

type handlerFactoryConfig struct {
	namespace                        string
	subsystem                        string
	requestDurationBuckets           []float64
	timeToWriteHeaderDurationBuckets []float64
	byteSizeBuckets                  []float64
	labels                           []string
}

// HandlerFactoryOption is used to pass options in NewHandlerFactory.
type HandlerFactoryOption func(*handlerFactoryConfig)

func applyHandlerFactoryOptions(opts []HandlerFactoryOption) handlerFactoryConfig {
	config := handlerFactoryConfig{
		subsystem: "http",
		requestDurationBuckets: []float64{
			0.005, /* 5ms */
			0.025, /* 25ms */
			0.1,   /* 100ms */
			0.5,   /* 500ms */
			1.0,   /* 1s */
			10.0,  /* 10s */
			30.0,  /* 30s */
			60.0,  /* 1m */
			300.0, /* 5m */
		},
		timeToWriteHeaderDurationBuckets: []float64{
			0.005, /* 5ms */
			0.025, /* 25ms */
			0.1,   /* 100ms */
			0.5,   /* 500ms */
			1.0,   /* 1s */
			10.0,  /* 10s */
			30.0,  /* 30s */
		},
		byteSizeBuckets: []float64{
			10,
			64,
			256,
			1024,             /* 1KiB */
			64 * 1024,        /* 64KiB */
			256 * 1024,       /* 256KiB */
			1024 * 1024,      /* 1MiB */
			64 * 1024 * 1024, /* 64MiB */
		},
		labels: []string{"code", "method"},
	}
	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithNamespace will configure the namespace to apply to the metrics.
func WithNamespace(namespace string) HandlerFactoryOption {
	return func(config *handlerFactoryConfig) {
		config.namespace = namespace
	}
}

// WithLabels will configure additional labels to apply to the metrics.
func WithLabels(labels ...string) HandlerFactoryOption {
	return func(config *handlerFactoryConfig) {
		config.labels = append(config.labels, labels...)
	}
}

// WithRequestDurationBuckets will configure the duration buckets used for
// incoming request histogram buckets.
func WithRequestDurationBuckets(buckets []float64) HandlerFactoryOption {
	return func(config *handlerFactoryConfig) {
		config.requestDurationBuckets = buckets
	}
}

// WithTimeToWriteHeaderDurationBuckets will configure the time to write header
// duration histogram buckets.
func WithTimeToWriteHeaderDurationBuckets(buckets []float64) HandlerFactoryOption {
	return func(config *handlerFactoryConfig) {
		config.timeToWriteHeaderDurationBuckets = buckets
	}
}

// WithByteSizeBuckets will configure the byte size histogram buckets for request
// and response payloads.
func WithByteSizeBuckets(buckets []float64) HandlerFactoryOption {
	return func(config *handlerFactoryConfig) {
		config.byteSizeBuckets = buckets
	}
}
