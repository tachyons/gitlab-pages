package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

var listenHTTP = flag.String("listen-http", ":80", "The address to listen for HTTP requests")
var listenHTTPS = flag.String("listen-https", "", "The address to listen for HTTPS requests")
var listenProxy = flag.String("listen-proxy", "", "The address to listen for proxy requests")
var pagesDomain = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")
var pagesRootCert = flag.String("root-cert", "", "The default path to file certificate to serve static pages")
var pagesRootKey = flag.String("root-key", "", "The default path to file certificate to serve static pages")
var serverHTTP = flag.Bool("serve-http", true, "Serve the pages under HTTP")
var http2proto = flag.Bool("http2", true, "Enable HTTP2 support")
var pagesRoot = flag.String("pages-root", "shared/pages", "The directory where pages are stored")

func resolve() {
	fullPath, err := filepath.EvalSymlinks(*pagesRoot)
	if err != nil {
		log.Fatalln(err)
	}
	*pagesRoot = fullPath
}

func main() {
	var wg sync.WaitGroup
	var app theApp

	fmt.Printf("GitLab Pages Daemon %s (%s)", VERSION, REVISION)
	fmt.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages")
	flag.Parse()
	resolve()

	// Listen for HTTP
	if *listenHTTP != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServe(*listenHTTP, app.ServeHTTP)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	// Listen for HTTPS
	if *listenHTTPS != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServeTLS(*listenHTTPS, *pagesRootCert, *pagesRootKey, app.ServeHTTP, app.ServeTLS)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	// Listen for HTTP proxy requests
	if *listenProxy != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServe(*listenProxy, app.ServeProxy)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	go watchDomains(app.UpdateDomains, time.Second)

	wg.Wait()
}
