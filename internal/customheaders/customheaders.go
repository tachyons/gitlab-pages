package customheaders

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

var (
	errInvalidHeaderParameter = errors.New("invalid syntax specified as header parameter")
	errDuplicateHeader        = errors.New("duplicate header")
)

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
	headers := make(http.Header, len(customHeaders))

	for _, h := range customHeaders {
		h = h + "\n\n"
		tp := textproto.NewReader(bufio.NewReader(strings.NewReader(h)))

		mimeHeader, err := tp.ReadMIMEHeader()
		if err != nil {
			return nil, fmt.Errorf("parsing error %s: %w", h, errInvalidHeaderParameter)
		}

		for key, value := range mimeHeader {
			if _, ok := headers[key]; ok {
				return nil, fmt.Errorf("%s already specified with value '%s': %w", key, value, errDuplicateHeader)
			}

			headers[key] = value
		}
	}

	return headers, nil
}
