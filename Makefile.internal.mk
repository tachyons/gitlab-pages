REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

ifneq (v$(VERSION),$(LAST_TAG))
	VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

VERSION_FLAGS := -ldflags='-X "main.VERSION=$(VERSION)" -X "main.REVISION=$(REVISION)"'

_allpackages = $(shell (go list ./... | \
	grep -v $(addprefix -e ,$(IGNORED_DIRS))))

# memoize allpackages, so that it's executed only once and only if used
allpackages = $(if $(__allpackages),,$(eval __allpackages := $$(_allpackages)))$(__allpackages)

export GOPATH := $(CURDIR)/.GOPATH
export GOBIN := $(CURDIR)/bin

Q := $(if $V,,@)

.GOPATH/.ok:
	mkdir -p .GOPATH

.PHONY: bin/gocovmerge bin/golangci-lint
bin/gocovmerge: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored gocovmerge not found, try running 'make setup'..."; exit 1; }

bin/golangci-lint: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored golangci-lint not found, try running 'make setup'..."; exit 1; }

bin/go-junit-report: .GOPATH/.ok
	@test -x $@ || \
	    { echo "Vendored go-junit-report not found, try running 'make setup'..."; exit 1; }
