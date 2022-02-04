BINDIR := $(CURDIR)/bin
GO_BUILD_TAGS   := continuous_profiler_stackdriver

# To compute a unique and deterministic value for GNU build-id, we build the Go binary a second time.
# From the first build, we extract its unique and deterministic Go build-id, and use that to derive
# a comparably unique and deterministic GNU build-id to inject into the final binary.
## Skip generation of the GNU build ID if set to speed up builds.
WITHOUT_BUILD_ID ?=

.PHONY: all setup generate-mocks build clean

all: gitlab-pages

setup: .GOPATH/.ok
	mkdir -p bin/
	# Installing dev tools defined in go.tools
	awk '/_/ {print $$2}' ./tools/main.go | xargs -tI % go install ${V:+-v -x} -modfile=tools/go.mod -mod=mod %

cisetup: .GOPATH/.ok
	mkdir -p bin/
	# Installing dev tools defined in go.tools
	awk '/_/ {print $$2}' ./tools/main.go | grep -v -e mockgen -e golangci | xargs -tI % go install ${V:+-v -x} -modfile=tools/go.mod -mod=mod %

generate-mocks: .GOPATH/.ok
	$Q bin/mockgen -source=internal/interface.go -destination=internal/handlers/mock/handler_mock.go -package=mock
	$Q bin/mockgen -source=internal/source/source.go -destination=internal/source/mock/source_mock.go -package=mock
	$Q bin/mockgen -source=internal/source/gitlab/mock/client_stub.go -destination=internal/source/gitlab/mock/client_mock.go -package=mock
	$Q bin/mockgen -source=internal/domain/resolver.go -destination=internal/domain/mock/resolver_mock.go -package=mock

build: .GOPATH/.ok
	$Q GOBIN=$(BINDIR) go install $(if $V,-v) -ldflags="$(VERSION_FLAGS)" -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)
ifndef WITHOUT_BUILD_ID
	GO_BUILD_ID=$$( go tool buildid $(BINDIR)/gitlab-pages ) && \
	GNU_BUILD_ID=$$( echo $$GO_BUILD_ID | sha1sum | cut -d' ' -f1 ) && \
	$Q GOBIN=$(BINDIR) go install $(if $V,-v) -ldflags="$(VERSION_FLAGS) -B 0x$$GNU_BUILD_ID" -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)
endif

clean:
	$Q GOBIN=$(BINDIR) go clean -i -modcache -x

gitlab-pages: build
	$Q cp -f $(BINDIR)/gitlab-pages .
