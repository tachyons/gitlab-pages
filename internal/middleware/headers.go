package middleware

import (
	"errors"
	"net/http"
	"strings"
)

var errInvalidHeaderParameter = errors.New("invalid syntax specified as header parameter")

// AddCustomHeaders adds a map of Headers to a Response
func AddCustomHeaders(w http.ResponseWriter, headers http.Header) {
	for k, v := range headers {
		for _, value := range v {
			w.Header().Add(k, value)
		}
	}
}

// ParseHeaderString parses a string of key values into a map
func ParseHeaderString(customHeaders []string) (http.Header, error) {
	headers := http.Header{}
	for _, keyValueString := range customHeaders {
		keyValue := strings.SplitN(keyValueString, ":", 2)
		if len(keyValue) != 2 {
			return nil, errInvalidHeaderParameter
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])

		headers[key] = append(headers[key], value)
	}
	return headers, nil
}
