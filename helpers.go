package main

import (
	"io/ioutil"
	"net"
	"strings"
)

func readFile(file string) (result []byte) {
	result, err := ioutil.ReadFile(file)
	if err != nil {
		fatal(err)
	}
	return
}

func createSocket(addr string) (l net.Listener, fd uintptr) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fatal(err)
	}

	f, err := l.(*net.TCPListener).File()
	if err != nil {
		fatal(err)
	}

	fd = f.Fd()
	return
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}
