package logging

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

// ConfigureLogging will initialize the system logger.
func ConfigureLogging(format string, verbose bool) error {
	var levelOption log.LoggerOption

	if format == "" {
		format = "json"
	}

	if verbose {
		levelOption = log.WithLogLevel("trace")
	} else {
		levelOption = log.WithLogLevel("info")
	}

	_, err := log.Initialize(
		log.WithFormatter(format),
		levelOption,
	)
	return err
}

// getAccessLogger will return the default logger, except when
// the log format is text, in which case a combined HTTP access
// logger will be configured. This behaviour matches Workhorse
func getAccessLogger(format string) (*logrus.Logger, error) {
	if format != "text" && format != "" {
		return logrus.StandardLogger(), nil
	}

	accessLogger := log.New()
	_, err := log.Initialize(
		log.WithLogger(accessLogger),  // Configure `accessLogger`
		log.WithFormatter("combined"), // Use the combined formatter
	)
	if err != nil {
		return nil, err
	}

	return accessLogger, nil
}

// getExtraLogFields is used to inject additional fields into the
// HTTP access logger middleware.
func getExtraLogFields(r *http.Request) log.Fields {
	logFields := log.Fields{
		"pages_https": request.IsHTTPS(r),
		"pages_host":  request.GetHost(r),
	}

	if d := request.GetDomain(r); d != nil {
		lp, err := d.GetLookupPath(r)
		if err != nil {
			logFields["error"] = err.Error()
			return logFields
		}

		logFields["pages_project_serving_type"] = lp.ServingType
		logFields["pages_project_prefix"] = lp.Prefix
		logFields["pages_project_id"] = lp.ProjectID
	}

	return logFields
}

// BasicAccessLogger configures the GitLab pages basic HTTP access logger middleware
func BasicAccessLogger(handler http.Handler, format string, extraFields log.ExtraFieldsGeneratorFunc) (http.Handler, error) {
	accessLogger, err := getAccessLogger(format)
	if err != nil {
		return nil, err
	}

	return log.AccessLogger(handler,
		log.WithExtraFields(enrichExtraFields(extraFields)),
		log.WithAccessLogger(accessLogger),
		// TODO: log IP for HTTP requests https://gitlab.com/gitlab-org/gitlab-pages/-/issues/640
		log.WithXFFAllowed(func(sip string) bool { return false }),
	), nil
}

func enrichExtraFields(extraFields log.ExtraFieldsGeneratorFunc) log.ExtraFieldsGeneratorFunc {
	return func(r *http.Request) log.Fields {
		enrichedFields := log.Fields{
			"correlation_id": correlation.ExtractFromContext(r.Context()),
			"pages_https":    request.IsHTTPS(r),
			"pages_host":     r.Host,
		}

		if extraFields != nil {
			for field, value := range extraFields(r) {
				enrichedFields[field] = value
			}
		}

		return enrichedFields
	}
}

// AccessLogger configures the GitLab pages HTTP access logger middleware with extra log fields
func AccessLogger(handler http.Handler, format string) (http.Handler, error) {
	return BasicAccessLogger(handler, format, getExtraLogFields)
}

// LogRequest will inject request host and path to the logged messages
func LogRequest(r *http.Request) *logrus.Entry {
	return log.WithFields(log.Fields{
		"correlation_id": correlation.ExtractFromContext(r.Context()),
		"host":           r.Host,
		"path":           r.URL.Path,
	})
}
