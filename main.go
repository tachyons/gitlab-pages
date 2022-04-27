package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"gitlab.com/gitlab-org/labkit/fips"
	"gitlab.com/gitlab-org/labkit/log"

	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/validateargs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func initErrorReporting(sentryDSN, sentryEnvironment string) error {
	return errortracking.Initialize(
		errortracking.WithSentryDSN(sentryDSN),
		errortracking.WithVersion(fmt.Sprintf("%s-%s", VERSION, REVISION)),
		errortracking.WithLoggerName("gitlab-pages"),
		errortracking.WithSentryEnvironment(sentryEnvironment))
}

func appMain() {
	if err := validateargs.NotAllowed(os.Args[1:]); err != nil {
		log.WithError(err).Fatal("Using invalid arguments, use -config=gitlab-pages-config file instead")
	}

	if err := validateargs.Deprecated(os.Args[1:]); err != nil {
		log.WithError(err).Warn("Using deprecated arguments")
	}

	config, err := cfg.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load config")
	}

	printVersion(config.General.ShowVersion, VERSION)

	if err := cfg.Validate(config); err != nil {
		log.WithError(err).Fatal("invalid config settings")
	}

	if config.Sentry.DSN != "" {
		err := initErrorReporting(config.Sentry.DSN, config.Sentry.Environment)
		if err != nil {
			log.WithError(err).Warn("Failed to initialize errortracking")
		}
	}

	err = logging.ConfigureLogging(config.Log.Format, config.Log.Verbose)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	cfg.LogConfig(config)

	log.WithFields(log.Fields{
		"version":  VERSION,
		"revision": REVISION,
	}).Info("GitLab Pages")
	log.Info("URL: https://gitlab.com/gitlab-org/gitlab-pages")

	if err := os.Chdir(config.General.RootDir); err != nil {
		fatal(err, "could not change directory into pagesRoot")
	}
	fips.Check()

	runApp(config)
}

func printVersion(showVersion bool, version string) {
	if showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		os.Exit(0)
	}
}

func main() {
	logrus.SetOutput(os.Stderr)

	rand.Seed(time.Now().UnixNano())

	metrics.MustRegister()

	appMain()
}
