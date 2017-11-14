package main

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func saveConfig(projectID int64, projectPath string, config map[string]interface{}) {
	data, err := json.MarshalIndent(config, "", "\t")
	panicOnError(err)

	if replaceFile(filepath.Join(*deployerRoot, projectPath), "config.json", data) {
		touchDaemon()
	}
}

func replaceFile(path, filename string, data []byte) bool {
	err := os.MkdirAll(path, 0750)
	panicOnFileSystemError(err)

	f, err := ioutil.TempFile(path, "config")
	panicOnError(err)
	defer f.Close()
	defer os.Remove(f.Name())

	println(f.Name())

	err = os.Rename(f.Name(), filepath.Join(path, filename))
	panicOnFileSystemError(err)
	return true
}

func touchDaemon() {
	randomData := make([]byte, 32)
	_, err := rand.Read(randomData)
	panicOnError(err)

	replaceFile(*deployerRoot, ".update", randomData)
}
