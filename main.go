package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func main() {
	var listenHTTP = flag.String("listen-http", ":80", "The address to listen for HTTP requests")
	var listenHTTPS = flag.String("listen-https", "", "The address to listen for HTTPS requests")
	var listenProxy = flag.String("listen-proxy", "", "The address to listen for proxy requests")
	var pagesRootCert = flag.String("root-cert", "", "The default path to file certificate to serve static pages")
	var pagesRootKey = flag.String("root-key", "", "The default path to file certificate to serve static pages")
	var redirectHTTP = flag.Bool("redirect-http", true, "Serve the pages under HTTP")
	var useHTTP2 = flag.Bool("use-http2", true, "Enable HTTP2 support")
	var pagesRoot = flag.String("pages-root", "shared/pages", "The directory where pages are stored")
	var pagesDomain = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")
	var pagesUser = flag.String("pages-user", "", "Drop privileges to this user")

	fmt.Printf("GitLab Pages Daemon %s (%s)\n", VERSION, REVISION)
	fmt.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages\n")
	flag.Parse()

	err := os.Chdir(*pagesRoot)
	if err != nil {
		log.Fatalln(err)
	}

	var app theApp
	app.Domain = strings.ToLower(*pagesDomain)
	app.RedirectHTTP = *redirectHTTP
	app.HTTP2 = *useHTTP2

	if *pagesRootCert != "" {
		app.RootCertificate = readFile(*pagesRootCert)
	}

	if *pagesRootKey != "" {
		app.RootKey = readFile(*pagesRootKey)
	}

<<<<<<< 9042f5171c4bddc3da330b0e236e5faa78e657c3
=======
	//daemonize()

	fmt.Println("Starting...")

	// We don't need root privileges any more
	//	if err := syscall.Setgid(33); err != nil {
	//		log.Fatalln("setgid:", err)
	//	}
	if err := syscall.Setuid(33); err != nil {
		log.Fatalln("setuid:", err)
	}

	err := syscall.Chroot(*pagesRoot)
	if err != nil {
		log.Fatalln("chroot:", err)
	}
	*pagesRoot = "/"

	// Listen for HTTP
>>>>>>> Daemonize
	if *listenHTTP != "" {
		var l net.Listener
		l, app.ListenHTTP = createSocket(*listenHTTP)
		defer l.Close()
	}

	if *listenHTTPS != "" {
		var l net.Listener
		l, app.ListenHTTPS = createSocket(*listenHTTPS)
		defer l.Close()
	}

	if *listenProxy != "" {
		var l net.Listener
		l, app.ListenHTTPS = createSocket(*listenProxy)
		defer l.Close()
	}

	app.Run()
}
