package main

import (
	"net/http"

	gitlab "github.com/xanzy/go-gitlab"
)

func panicOnAPIError(resp *gitlab.Response, err error) {
	if resp != nil {
		panicOnHTTPError(resp.Response, err)
	} else {
		panicOnHTTPError(nil, err)
	}
}

func panicOnHTTPError(resp *http.Response, err error) {
	if err == nil {
		return
	}
	if resp != nil {
		if resp.StatusCode/100 == 2 {
			return
		}
		if resp.StatusCode == http.StatusNotFound {
			return
		}
		if resp.StatusCode == http.StatusBadRequest {
			return
		}
	}
	panic(err)
}
