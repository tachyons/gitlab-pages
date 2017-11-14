package main

import (
	"io/ioutil"
	"log"
	"net"
	"strings"
)

func readFile(file string) (result []byte) {
	result, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func createSocket(addr string) (l net.Listener, fd uintptr) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := l.(*net.TCPListener).File()
	if err != nil {
		log.Fatalln(err)
	}

	fd = f.Fd()
	return
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}
