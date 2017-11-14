package main

import workers "github.com/jrallison/go-workers"

func renameNamespaceJob(message *workers.Msg, projectId int64, fullPathWas, fullPath string) {

}

func renameProjectJob(message *workers.Msg, projectId int64, pathWas, path, fullPath string) {

}

func moveProjectJob(message *workers.Msg, projectId int64, path, fullPathWas, fullPath string) {

}

func removeJob(message *workers.Msg, projectId int64, namespacePath, path string) {

}
