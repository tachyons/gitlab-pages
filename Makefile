REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)

ifneq ($(VERSION),$(LAST_TAG))
    VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

GO_LDFLAGS ?= -X main.VERSION=$(VERSION) -X main.REVISION=$(REVISION)
GO_FILES ?= $(shell find . -name '*.go')

export GO15VENDOREXPERIMENT := 1

all: gitlab-pages

gitlab-pages: $(GO_FILES)
	go build -o gitlab-pages --ldflags="$(GO_LDFLAGS)"

update:
	godep save ./...

verify: fmt vet lint complexity test

fmt:
	go fmt ./... | awk '{ print "Please run go fmt"; exit 1 }'

vet:
	go get golang.org/x/tools/cmd/vet
	go vet

lint:
	go get github.com/golang/lint/golint
	golint . | ( ! grep -v "^$$" )

complexity:
	go get github.com/fzipp/gocyclo
	gocyclo -over 8 $(wildcard *.go)

test:
	go get golang.org/x/tools/cmd/cover
	go test ./... -cover
