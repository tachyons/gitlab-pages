REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell cat VERSION)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

ifneq (v$(VERSION),$(LAST_TAG))
	VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif

VERSION_FLAGS :=-X "main.VERSION=$(VERSION)" -X "main.REVISION=$(REVISION)"

export GOBIN := $(CURDIR)/bin

Q := $(if $V,,@)
