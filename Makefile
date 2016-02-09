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
