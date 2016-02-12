package main

import (
	"log"
	"os"
)

var chdirSet = false

func setUpTests() {
	if chdirSet {
		return
	}

	err := os.Chdir("shared/pages")
	if err != nil {
		log.Println("Chdir:", err)
	} else {
		chdirSet = true
	}
}
