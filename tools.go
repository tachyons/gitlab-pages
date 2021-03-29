//+build tools

package main

// These imports are to force `go mod tidy` not to remove that tools we depend
// on development. This is explained in great detail in
// https://marcofranssen.nl/manage-go-tools-via-go-modules/
import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/wadey/gocovmerge"
	_ "golang.org/x/tools/cmd/goimports"
)
