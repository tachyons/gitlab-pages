package config

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
	return strings.Join(s.value, s.sep())
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
	for _, str := range s.value {
		result = append(result, strings.Split(str, s.sep())...)
	}

	return
}

func (s *MultiStringFlag) sep() string {
	if s.separator == "" {
		return defaultSeparator
	}

	return s.separator
}

func (s *MultiStringFlag) Len() int {
	return len(s.value)
}
