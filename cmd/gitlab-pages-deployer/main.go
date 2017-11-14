package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	workers "github.com/jrallison/go-workers"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

var (
	showVersion = flag.Bool("version", false, "Show version")

	apiURL         = flag.String("api-url", "https://gitlab.com/api/v4", "The API URL to GitLab")
	apiAccessToken = flag.String("api-access-token", "", "API Access Token to post back the statuses of Pages")

	statsServer = flag.String("stats-server", "localhost:9818", "Address to statistics server")

	deployerRoot        = flag.String("deployer-root", "shared/pages", "The directory where pages are stored")
	deployerMaximumSize = flag.Int("deployer-maximum-size", 1024, "The maximum size of artifacts extracted (in Bytes)")
	deployerConcurrency = flag.Int("deployer-concurrency", 10, "The maximum concurrency")
	deployerID          = flag.String("deployer-id", "1", "Unique ID of Deployer")

	redisServer    = flag.String("redis-server", "localhost:6379", "The address of Redis Server")
	redisDatabase  = flag.String("redis-database", "0", "The name of Redis Database")
	redisPassword  = flag.String("redis-password", "", "The password to Redis Database")
	redisNamespace = flag.String("redis-namespace", "", "The namespace to use")
	redisPool      = flag.Int("redis-password", 10, "The connection pool to Redis Database")
)

func runStatsServer() {
	http.HandleFunc("/stats", workers.Stats)

	if *statsServer != "" {
		log.Println("Stats are available at", fmt.Sprint(*statsServer, "/stats"))

		go func() {
			if err := http.ListenAndServe(*statsServer, nil); err != nil {
				log.Println(err)
			}
		}()
	}
}

func appMain() {
	flag.Parse()

	printVersion(*showVersion, VERSION)

	log.Printf("GitLab Pages Deployer %s (%s)", VERSION, REVISION)
	log.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages\n")

	runStatsServer()

	workers.Configure(map[string]string{
		"server":   *redisServer,
		"database": *redisDatabase,
		"password": *redisPassword,
		"pool":     strconv.Itoa(*redisPool),
		"process":  *deployerID,
	})

	workers.Middleware.Append(&myMiddleware{})
	workers.Process("pages", pagesJob, *deployerConcurrency)
	workers.Run()
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

	appMain()
}
