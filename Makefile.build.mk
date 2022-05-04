BINDIR := $(CURDIR)/bin
GO_BUILD_TAGS   := continuous_profiler_stackdriver
MOCKGEN_VERSION=v1.6.0
FIPS_MODE       ?= 0
ifeq ($(FIPS_MODE), 1)
    GO_BUILD_TAGS := $(GO_BUILD_TAGS),fips
endif

# To compute a unique and deterministic value for GNU build-id, we build the Go binary a second time.
# From the first build, we extract its unique and deterministic Go build-id, and use that to derive
# a comparably unique and deterministic GNU build-id to inject into the final binary.
## Skip generation of the GNU build ID if set to speed up builds.
WITHOUT_BUILD_ID ?=

.PHONY: all generate-mocks build clean

all: gitlab-pages

generate-mocks:
	$Q go run github.com/golang/mock/mockgen@$(MOCKGEN_VERSION) -source=internal/interface.go -destination=internal/handlers/mock/handler_mock.go -package=mock
	$Q go run github.com/golang/mock/mockgen@$(MOCKGEN_VERSION) -source=internal/source/source.go -destination=internal/source/mock/source_mock.go -package=mock
	$Q go run github.com/golang/mock/mockgen@$(MOCKGEN_VERSION) -source=internal/source/gitlab/mock/client_stub.go -destination=internal/source/gitlab/mock/client_mock.go -package=mock
	$Q go run github.com/golang/mock/mockgen@$(MOCKGEN_VERSION) -source=internal/domain/resolver.go -destination=internal/domain/mock/resolver_mock.go -package=mock

build:
	$Q GOBIN=$(BINDIR) go install $(if $V,-v) -ldflags="$(VERSION_FLAGS)" -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)
ifndef WITHOUT_BUILD_ID
	GO_BUILD_ID=$$( go tool buildid $(BINDIR)/gitlab-pages ) && \
	GNU_BUILD_ID=$$( echo $$GO_BUILD_ID | sha1sum | cut -d' ' -f1 ) && \
	$Q GOBIN=$(BINDIR) go install $(if $V,-v) -ldflags="$(VERSION_FLAGS) -B 0x$$GNU_BUILD_ID" -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)
endif
ifeq ($(FIPS_MODE), 1)
	go tool nm $(BINDIR)/gitlab-pages | grep boringcrypto >/dev/null &&  echo "binary is correctly built in FIPS mode" || (echo "binary is not correctly built in FIPS mode" && exit 1)
endif

clean:
	$Q GOBIN=$(BINDIR) go clean -i -modcache -x

gitlab-pages: build
	$Q cp -f $(BINDIR)/gitlab-pages .
