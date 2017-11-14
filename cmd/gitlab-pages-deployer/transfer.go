package main

import (
	"os"
	"path/filepath"

	workers "github.com/jrallison/go-workers"
)

func panicOnFileSystemError(err error) {
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
}

func renameNamespaceJob(message *workers.Msg, projectID int64, fullPathWas, fullPath string) {
	err := os.Rename(filepath.Join(*deployerRoot, fullPathWas), filepath.Join(*deployerRoot, fullPath))
	panicOnFileSystemError(err)

	touchDaemon()
}

func renameProjectJob(message *workers.Msg, projectID int64, pathWas, path, fullPath string) {
	err := os.Rename(filepath.Join(*deployerRoot, fullPath, pathWas), filepath.Join(*deployerRoot, fullPath, path))
	panicOnFileSystemError(err)

	touchDaemon()
}

func moveProjectJob(message *workers.Msg, projectID int64, path, fullPathWas, fullPath string) {
	err := os.MkdirAll(filepath.Join(*deployerRoot, fullPath), 0750)
	panicOnFileSystemError(err)

	err = os.Rename(filepath.Join(*deployerRoot, fullPathWas, path), filepath.Join(*deployerRoot, fullPath, path))
	panicOnFileSystemError(err)

	touchDaemon()
}

func removeJob(message *workers.Msg, projectID int64, namespacePath, path string) {
	err := os.RemoveAll(filepath.Join(*deployerRoot, namespacePath, path))
	panicOnFileSystemError(err)

	touchDaemon()
}
