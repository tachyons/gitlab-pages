package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	accessLogFormat = "text"
	logrusEntry     = log.WithField("system", "http")
)

func configureLogging(format string, verbose bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	switch format {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
		accessLogFormat = "json"
	default:
		log.SetFormatter(&log.TextFormatter{})
		accessLogFormat = "text"
	}
}

type loggingResponseWriter struct {
	rw      http.ResponseWriter
	status  int
	written int64
	started time.Time
}

func newLoggingResponseWriter(rw http.ResponseWriter) loggingResponseWriter {
	return loggingResponseWriter{
		rw:      rw,
		started: time.Now(),
	}
}

func (l *loggingResponseWriter) Header() http.Header {
	return l.rw.Header()
}

func (l *loggingResponseWriter) Write(data []byte) (n int, err error) {
	if l.status == 0 {
		l.WriteHeader(http.StatusOK)
	}
	n, err = l.rw.Write(data)
	l.written += int64(n)
	return
}

func (l *loggingResponseWriter) WriteHeader(status int) {
	if l.status != 0 {
		return
	}

	l.status = status
	l.rw.WriteHeader(status)
}

func (l *loggingResponseWriter) extractLogFields(r *http.Request) log.Fields {
	referer := r.Referer()
	parsedReferer, err := url.Parse(referer)

	// The referer query string may contain credentials, so remove if possible
	if err == nil {
		parsedReferer.RawQuery = ""
		referer = parsedReferer.String()
	}

	return log.Fields{
		"host":       r.Host,
		"remoteAddr": r.RemoteAddr,
		"method":     r.Method,
		"uri":        r.URL.Path, //The request query string may contain credentials
		"proto":      r.Proto,
		"status":     l.status,
		"written":    l.written,
		"referer":    referer,
		"userAgent":  r.UserAgent(),
		"duration":   time.Since(l.started).Seconds(),
	}
}

func (l *loggingResponseWriter) Log(r *http.Request) {
	fields := l.extractLogFields(r)

	switch accessLogFormat {
	case "text":
		fmt.Printf("%s %s - - [%s] %q %d %d %q %q %f\n",
			fields["host"], fields["remoteAddr"], l.started,
			fmt.Sprintf("%s %s %s", fields["method"], fields["uri"], fields["proto"]),
			fields["status"], fields["written"], fields["referer"], fields["userAgent"], fields["duration"],
		)
	case "json":
		logrusEntry.WithFields(fields).Info("access")
	default:
		panic("invalid access log format")
	}
}

func fatal(err error) {
	log.WithError(err).Fatal()
}
