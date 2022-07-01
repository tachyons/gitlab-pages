package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/test/gitlabstub"
)

var (
	pagesRoot = flag.String("pages-root", "shared/pages", "The directory where pages are stored")
	keyFile   = flag.String("key-file", "", "Path to file certificate")
	certFile  = flag.String("cert-file", "", "Path to file certificate")
)

func main() {
	flag.Parse()

	var opts []gitlabstub.Option

	if *keyFile != "" && *certFile != "" {
		log.Printf("Loading key pair: (%s) - (%s)", *certFile, *keyFile)
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("error loading certificate: %v", err)
		}

		opts = append(opts, gitlabstub.WithCertificate(cert))
	}

	if err := os.Chdir(*pagesRoot); err != nil {
		log.Fatalf("error chdir in %s: %v", *pagesRoot, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current dir: %v", err)
	}

	opts = append(opts, gitlabstub.WithPagesRoot(wd))

	server, err := gitlabstub.NewUnstartedServer(opts...)
	if err != nil {
		log.Fatalf("error starting the server: %v", err)
	}

	if server.TLS != nil {
		server.StartTLS()
	} else {
		server.Start()
	}

	log.Printf("listening on %s\n", server.URL)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Config.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("error shutting down %v", err)
	}
}
