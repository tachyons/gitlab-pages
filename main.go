package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func evalSymlinks(directory string) (result string) {
	result, err := filepath.EvalSymlinks(directory)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func readFile(file string) (result []byte) {
	result, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func createSocket(addr string) (l net.Listener, fd uintptr) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := l.(*net.TCPListener).File()
	if err != nil {
		log.Fatalln(err)
	}

	fd = f.Fd()
	return
}

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

	fmt.Printf("GitLab Pages Daemon %s (%s)", VERSION, REVISION)
	fmt.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages")
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
}
