package main

import (
	"errors"
	"strings"
)

var errMultiStringSetEmptyValue = errors.New("value cannot be empty")

const defaultSeparator = ","

// MultiStringFlag implements the flag.Value interface and allows a string flag
// to be specified multiple times on the command line.
//
// e.g.: -listen-http 127.0.0.1:80 -listen-http [::1]:80
type MultiStringFlag struct {
	value     []string
	separator string
}

// String returns the list of parameters joined with a commas (",")
func (s *MultiStringFlag) String() string {
	if s.separator == "" {
		s.separator = defaultSeparator
	}

	return strings.Join(s.value, s.separator)
}

// Set appends the value to the list of parameters
func (s *MultiStringFlag) Set(value string) error {
	if value == "" {
		return errMultiStringSetEmptyValue
	}

	s.value = append(s.value, value)
	return nil
}

// Split each flag
func (s *MultiStringFlag) Split() (result []string) {
	if s.separator == "" {
		s.separator = defaultSeparator
	}

	for _, str := range s.value {
		result = append(result, strings.Split(str, s.separator)...)
	}

	return
}
