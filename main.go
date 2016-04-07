package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func appMain() {
	var showVersion = flag.Bool("version", false, "Show version")
	var listenHTTP = flag.String("listen-http", "", "The address to listen for HTTP requests")
	var listenHTTPS = flag.String("listen-https", "", "The address to listen for HTTPS requests")
	var listenProxy = flag.String("listen-proxy", "", "The address to listen for proxy requests")
	var pagesRootCert = flag.String("root-cert", "", "The default path to file certificate to serve static pages")
	var pagesRootKey = flag.String("root-key", "", "The default path to file certificate to serve static pages")
	var redirectHTTP = flag.Bool("redirect-http", true, "Serve the pages under HTTP")
	var useHTTP2 = flag.Bool("use-http2", true, "Enable HTTP2 support")
	var pagesRoot = flag.String("pages-root", "shared/pages", "The directory where pages are stored")
	var pagesDomain = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")
	var daemonUID = flag.Uint("daemon-uid", 0, "Drop privileges to this user")
	var daemonGID = flag.Uint("daemon-gid", 0, "Drop privileges to this group")

	flag.Parse()

	printVersion(*showVersion, VERSION)

	log.Printf("GitLab Pages Daemon %s (%s)", VERSION, REVISION)
	log.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages\n")

	err := os.Chdir(*pagesRoot)
	if err != nil {
		log.Fatalln(err)
	}

	var config appConfig
	config.Domain = strings.ToLower(*pagesDomain)
	config.RedirectHTTP = *redirectHTTP
	config.HTTP2 = *useHTTP2

	if *pagesRootCert != "" {
		config.RootCertificate = readFile(*pagesRootCert)
	}

	if *pagesRootKey != "" {
		config.RootKey = readFile(*pagesRootKey)
	}

	if *listenHTTP != "" {
		var l net.Listener
		l, config.ListenHTTP = createSocket(*listenHTTP)
		defer l.Close()
	}

	if *listenHTTPS != "" {
		var l net.Listener
		l, config.ListenHTTPS = createSocket(*listenHTTPS)
		defer l.Close()
	}

	if *listenProxy != "" {
		var l net.Listener
		l, config.ListenProxy = createSocket(*listenProxy)
		defer l.Close()
	}

	if *daemonUID != 0 || *daemonGID != 0 {
		daemonize(config, *daemonUID, *daemonGID)
		return
	}

	runApp(config)
}

func printVersion(showVersion bool, version string) {
	if showVersion {
		log.SetFlags(0)
		log.Printf(version)
		os.Exit(0)
	}
}

func main() {
	log.SetOutput(os.Stderr)

	daemonMain()
	appMain()
}
