package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func appMain() {
	var showVersion = flag.Bool("version", false, "Show version")
	var listenHTTP, listenHTTPS, listenProxy MultiStringFlag

	flag.Var(&listenHTTP, "listen-http", "The address(es) to listen on for HTTP requests")
	flag.Var(&listenHTTPS, "listen-https", "The address(es) to listen on for HTTPS requests")
	flag.Var(&listenProxy, "listen-proxy", "The address(es) to listen on for proxy requests")

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

	for _, addr := range listenHTTP {
		l, fd := createSocket(addr)
		defer l.Close()
		config.ListenHTTP = append(config.ListenHTTP, fd)
	}

	for _, addr := range listenHTTPS {
		l, fd := createSocket(addr)
		defer l.Close()
		config.ListenHTTPS = append(config.ListenHTTPS, fd)
	}

	for _, addr := range listenProxy {
		l, fd := createSocket(addr)
		defer l.Close()
		config.ListenProxy = append(config.ListenProxy, fd)
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
