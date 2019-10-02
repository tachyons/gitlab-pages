package logging

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

// ConfigureLogging will initialize the system logger.
func ConfigureLogging(format string, verbose bool) error {
	var levelOption log.LoggerOption

	if format == "" {
		format = "text"
	}

	if verbose {
		levelOption = log.WithLogLevel("debug")
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
	var projectID uint64
	if d := request.GetDomain(r); d != nil {
		projectID = d.GetProjectID(r)
	}

	return log.Fields{
		"pages_https":      request.IsHTTPS(r),
		"pages_host":       request.GetHost(r),
		"pages_project_id": projectID,
	}
}

// AccessLogger configures the GitLab pages HTTP access logger middleware
func AccessLogger(handler http.Handler, format string) (http.Handler, error) {

	accessLogger, err := getAccessLogger(format)
	if err != nil {
		return nil, err
	}

	return log.AccessLogger(handler,
		log.WithExtraFields(getExtraLogFields),
		log.WithAccessLogger(accessLogger),
	), nil
}

// LogRequest will inject request host and path to the logged messages
func LogRequest(r *http.Request) *logrus.Entry {
	return log.WithFields(log.Fields{
		"host": r.Host,
		"path": r.URL.Path,
	})
}
