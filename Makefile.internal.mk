REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)

ifneq (v$(VERSION),$(LAST_TAG))
	VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

VERSION_FLAGS := -ldflags='-X "main.VERSION=$(VERSION)" -X "main.REVISION=$(REVISION)"'

# cd into the GOPATH to workaround ./... not following symlinks
_allpackages = $(shell ( cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && \
	GOPATH=$(CURDIR)/.GOPATH go list ./... 2>&1 1>&3 | \
	grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) 1>&2 ) 3>&1 | \
	grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)))

_allfiles = $(shell cd $(CURDIR)/.GOPATH/src/$(IMPORT_PATH) && find . -iname '*.go' | grep -v "^\./\." | grep -v -e "^$$" $(addprefix -e ,$(IGNORED_PACKAGES)) )

# memoize allpackages, so that it's executed only once and only if used
allpackages = $(if $(__allpackages),,$(eval __allpackages := $$(_allpackages)))$(__allpackages)
allfiles = $(if $(__allfiles),,$(eval __allfiles  := $$(_allfiles)))$(__allfiles)

export GOPATH := $(CURDIR)/.GOPATH
unexport GOBIN

Q := $(if $V,,@)

.GOPATH/.ok:
	$Q mkdir -p "$(dir .GOPATH/src/$(IMPORT_PATH))"
	$Q ln -s ../../../.. ".GOPATH/src/$(IMPORT_PATH)"
	$Q mkdir -p .GOPATH/test .GOPATH/cover
	$Q mkdir -p bin
	$Q ln -s ../bin .GOPATH/bin
	$Q touch $@

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
