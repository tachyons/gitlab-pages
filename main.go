package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/labkit/errortracking"

	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/validateargs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func initErrorReporting(sentryDSN, sentryEnvironment string) {
	errortracking.Initialize(
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

	config := cfg.LoadConfig()

	printVersion(config.General.ShowVersion, VERSION)

	if config.Sentry.DSN != "" {
		initErrorReporting(config.Sentry.DSN, config.Sentry.Environment)
	}

	err := logging.ConfigureLogging(config.Log.Format, config.Log.Verbose)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	cfg.LogConfig(config)

	log.WithFields(log.Fields{
		"version":  VERSION,
		"revision": REVISION,
	}).Print("GitLab Pages Daemon")
	log.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages")

	if config.General.RootDir == "false" {
		log.Info("pages-root is disabled!")
	} else if err := os.Chdir(config.General.RootDir); err != nil {
		fatal(err, "could not change directory into pagesRoot")
	}

	for _, cs := range [][]io.Closer{
		createAppListeners(config),
		createMetricsListener(config),
	} {
		defer closeAll(cs)
	}

	if config.Daemon.UID != 0 || config.Daemon.GID != 0 {
		if err := daemonize(config); err != nil {
			errortracking.Capture(err)
			fatal(err, "could not create pages daemon")
		}

		return
	}

	runApp(config)
}

func closeAll(cs []io.Closer) {
	for _, c := range cs {
		c.Close()
	}
}

// createAppListeners returns net.Listener and *os.File instances. The
// caller must ensure they don't get closed or garbage-collected (which
// implies closing) too soon.
func createAppListeners(config *cfg.Config) []io.Closer {
	var closers []io.Closer
	var httpListeners []uintptr
	var httpsListeners []uintptr
	var proxyListeners []uintptr
	var httpsProxyv2Listeners []uintptr

	for _, addr := range config.ListenHTTPStrings.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up HTTP listener")

		httpListeners = append(httpListeners, f.Fd())
	}

	for _, addr := range config.ListenHTTPSStrings.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up HTTPS listener")

		httpsListeners = append(httpsListeners, f.Fd())
	}

	for _, addr := range config.ListenProxyStrings.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up proxy listener")

		proxyListeners = append(proxyListeners, f.Fd())
	}

	for _, addr := range config.ListenHTTPSProxyv2Strings.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up https proxyv2 listener")

		httpsProxyv2Listeners = append(httpsProxyv2Listeners, f.Fd())
	}

	config.Listeners = cfg.Listeners{
		HTTP:         httpListeners,
		HTTPS:        httpsListeners,
		Proxy:        proxyListeners,
		HTTPSProxyv2: httpsProxyv2Listeners,
	}

	return closers
}

// createMetricsListener returns net.Listener and *os.File instances. The
// caller must ensure they don't get closed or garbage-collected (which
// implies closing) too soon.
func createMetricsListener(config *cfg.Config) []io.Closer {
	addr := config.General.MetricsAddress
	if addr == "" {
		return nil
	}

	l, f := createSocket(addr)
	config.ListenMetrics = f.Fd()

	log.WithFields(log.Fields{
		"listener": addr,
	}).Debug("Set up metrics listener")

	return []io.Closer{l, f}
}

func printVersion(showVersion bool, version string) {
	if showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		os.Exit(0)
	}
}

func main() {
	log.SetOutput(os.Stderr)

	rand.Seed(time.Now().UnixNano())

	metrics.MustRegister()

	daemonMain()
	appMain()
}
