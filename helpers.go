package main

import (
	"io/ioutil"
	"net"
	"os"
)

func readFile(file string) (result []byte) {
	result, err := ioutil.ReadFile(file)
	if err != nil {
		fatal(err)
	}
	return
}

// Be careful: if you let either of the return values get garbage
// collected by Go they will be closed automatically.
func createSocket(addr string) (net.Listener, *os.File) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fatal(err)
	}

	return l, fileForListener(l)
}

// Be careful: if you let either of the return values get garbage
// collected by Go they will be closed automatically.
func createUnixSocket(addr string) (net.Listener, *os.File) {
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		fatal(err)
	}

	l, err := net.Listen("unix", addr)
	if err != nil {
		fatal(err)
	}

	// This socket should be world-accessible; we have authentication at the
	// application level. When pages runs with privilege separation, the
	// default permissions will prevent gitlab-rails from connecting to the
	// admin socket.
	if err := os.Chmod(addr, 0777); err != nil {
		fatal(err)
	}

	return l, fileForListener(l)
}

func fileForListener(l net.Listener) *os.File {
	type filer interface {
		File() (*os.File, error)
	}

	f, err := l.(filer).File()
	if err != nil {
		fatal(err)
	}

	return f
}
