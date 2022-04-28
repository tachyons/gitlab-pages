REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

ifneq (v$(VERSION),$(LAST_TAG))
	VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

VERSION_FLAGS :=-X "main.VERSION=$(VERSION)" -X "main.REVISION=$(REVISION)"

export GOPATH := $(CURDIR)/.GOPATH
export GOBIN := $(CURDIR)/bin

Q := $(if $V,,@)

.GOPATH/.ok:
	mkdir -p .GOPATH

.PHONY: bin/golangci-lint
bin/golangci-lint: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored golangci-lint not found, try running 'make setup'..."; exit 1; }

bin/gotestsum: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored gotestsum not found, try running 'make setup'..."; exit 1; }

bin/mockgen: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored mockgen not found, try running 'make setup'..."; exit 1; }
