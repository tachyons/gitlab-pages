package main

import (
	"strings"
)

// MultiStringFlag implements the flag.Value interface and allows a string flag
// to be specified multiple times on the command line.
//
// e.g.: -listen-http 127.0.0.1:80 -listen-http [::1]:80
type MultiStringFlag []string

// String returns the list of parameters joined with a commas (",")
func (s *MultiStringFlag) String() string {
	return strings.Join(*s, ",")
}

// Set appends the value to the list of parameters
func (s *MultiStringFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Split each flag
func (s *MultiStringFlag) Split() (result []string) {
	for _, str := range *s {
		result = append(result, strings.Split(str, ",")...)
	}

	return
}
