package metrics

type handlerConfig struct {
	labelValues map[string]string
}

// HandlerOption is used to pass options to the HandlerFactory instance.
type HandlerOption func(*handlerConfig)

func applyHandlerOptions(opts []HandlerOption) handlerConfig {
	config := handlerConfig{}
	for _, v := range opts {
		v(&config)
	}

	return config
}

// WithLabelValues will configure labels values to apply to this handler.
func WithLabelValues(labelValues map[string]string) HandlerOption {
	return func(config *handlerConfig) {
		config.labelValues = labelValues
	}
}
