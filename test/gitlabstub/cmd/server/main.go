package main

import (
	"context"
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
)

func main() {
	flag.Parse()

	if err := os.Chdir(*pagesRoot); err != nil {
		log.Fatalf("error chdir in %s: %v", *pagesRoot, err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current dir: %v", err)
	}

	server, err := gitlabstub.NewUnstartedServer(gitlabstub.WithPagesRoot(wd))
	if err != nil {
		log.Fatalf("error starting the server: %v", err)
	}

	server.Start()

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
