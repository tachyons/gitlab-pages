package main

import (
	log "github.com/sirupsen/logrus"
)

var (
	accessLogFormat = "text"
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

func fatal(err error) {
	log.WithError(err).Fatal()
}
