REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)

ifneq (v$(VERSION),$(LAST_TAG))
    VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

GO_LDFLAGS ?= -X main.VERSION=$(VERSION) -X main.REVISION=$(REVISION)
GO_FILES ?= $(shell find . -name '*.go')

export GO15VENDOREXPERIMENT := 1
export CGO_ENABLED := 0

all: gitlab-pages

gitlab-pages: $(GO_FILES)
	go build -o gitlab-pages --ldflags="$(GO_LDFLAGS)"

update:
	godep save ./...

verify-lite: fmt vet complexity test # lint does not work on go1.5 any more
verify: verify-lite lint

fmt:
	go fmt ./... | awk '{ print "Please run go fmt"; exit 1 }'

vet:
	go tool vet *.go

lint:
	go get github.com/golang/lint/golint
	golint . | ( ! grep -v "^$$" )

complexity:
	go get github.com/fzipp/gocyclo
	gocyclo -over 9 $(wildcard *.go)

test:
	go get golang.org/x/tools/cmd/cover
	go test . -short -cover -v -timeout 1m

acceptance: gitlab-pages
	go get golang.org/x/tools/cmd/cover
	go test . -cover -v -timeout 1m

docker:
	docker run --rm -it -v ${PWD}:/go/src/pages -w /go/src/pages golang:1.5 /bin/bash
