package singlehost

import (
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

type responseWriter struct {
	http.ResponseWriter
	pagesDomain string
}

func newResponseWriter(original http.ResponseWriter, pagesDomain string) http.ResponseWriter {
	return responseWriter{ResponseWriter: original, pagesDomain: pagesDomain}
}

func (w responseWriter) WriteHeader(statusCode int) {
	if statusCode == http.StatusMovedPermanently || statusCode == http.StatusFound {
		header := w.ResponseWriter.Header()
		header.Set("Location", w.transformLocation(header.Get("Location")))
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w responseWriter) transformLocation(location string) string {
	URL, err := url.Parse(location)
	if err != nil {
		log.WithField("location", location).WithError(err).Error("Can't parse redirected location")

		return location
	}

	if !strings.HasSuffix(URL.Hostname(), "."+w.pagesDomain) {
		log.WithFields(log.Fields{
			"hostname":     URL.Hostname(),
			"pages_domain": w.pagesDomain,
		}).Debug("Redirected URL doesn't match pages domain")

		return location
	}

	namespace := strings.TrimSuffix(URL.Hostname(), "."+w.pagesDomain)

	host := w.pagesDomain

	if URL.Port() != "" {
		host += ":" + URL.Port()
	}

	URL.Host = host
	URL.Path = "/" + namespace + URL.Path

	newLocation := URL.String()

	log.WithFields(log.Fields{
		"orig_location": location,
		"new_location":  newLocation,
	}).Debug("Changing redirected location")

	return newLocation
}
