package main

import workers "github.com/jrallison/go-workers"

func pagesJob(message *workers.Msg) {
	println(message.ToJson())
}
