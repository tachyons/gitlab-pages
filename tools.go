//+build tools

package main

import (
	_ "github.com/fzipp/gocyclo"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/wadey/gocovmerge"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
)
