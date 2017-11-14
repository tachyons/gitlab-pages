package main

import (
	"bytes"
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
	targetFile := filepath.Join(path, filename)

	err := os.MkdirAll(path, 0750)
	panicOnFileSystemError(err)

	dataWas, err := ioutil.ReadFile(targetFile)
	if err == nil && bytes.Equal(data, dataWas) {
		return false
	}

	f, err := ioutil.TempFile(path, "config")
	panicOnError(err)
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write(data)
	panicOnError(err)

	err = os.Rename(f.Name(), targetFile)
	panicOnFileSystemError(err)
	return true
}

func touchDaemon() {
	randomData := make([]byte, 32)
	_, err := rand.Read(randomData)
	panicOnError(err)

	replaceFile(*deployerRoot, ".update", randomData)
}
