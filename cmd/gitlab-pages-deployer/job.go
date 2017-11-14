package main

import (
	"time"

	workers "github.com/jrallison/go-workers"
)

func pagesJob(message *workers.Msg) {
	time.Sleep(time.Minute)
	println(message.Args())
}
