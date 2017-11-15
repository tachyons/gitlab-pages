package main

import (
	workers "github.com/jrallison/go-workers"
)

func pagesJob(message *workers.Msg) {
	if message.Get("class").MustString() != "PagesWorker" {
		panic("Expected PagesWorker class: " + message.Get("class").MustString())
	}

	args := message.Args()

	switch args.GetIndex(0).MustString() {
	case "deploy":
		deployJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustInt(),
			args.GetIndex(4).MustInt(),
			args.GetIndex(5).MustMap(),
		)

	case "remove":
		removeJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustString(),
		)

	case "config":
		configJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustMap(),
		)

	case "rename_namespace":
		renameNamespaceJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustString(),
		)

	case "rename_project":
		renameProjectJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustString(),
			args.GetIndex(4).MustString(),
		)

	case "move_project":
		moveProjectJob(
			message,
			args.GetIndex(1).MustInt(),
			args.GetIndex(2).MustString(),
			args.GetIndex(3).MustString(),
			args.GetIndex(4).MustString(),
		)

	default:
		panic("Unknown method: " + args.GetIndex(0).MustString())
	}
}
