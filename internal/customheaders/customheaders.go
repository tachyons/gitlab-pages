package customheaders

import (
	"bufio"
	"errors"
	"net/http"
	"net/textproto"
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
		keyValueString = strings.TrimSpace(keyValueString) + "\n\n"
		tp := textproto.NewReader(bufio.NewReader(strings.NewReader(keyValueString)))
		keyValue, err := tp.ReadMIMEHeader()
		if err != nil {
			return nil, errInvalidHeaderParameter
		}

		for k, v := range keyValue {
			k = textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(k))
			headers[k] = append(headers[k], v...)
		}
	}
	return headers, nil
}
