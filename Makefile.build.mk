GO_BUILD_TAGS   := continuous_profiler_stackdriver

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
	$Q bin/mockgen -source=internal/interface.go -destination=internal/mocks/mocks.go -package=mocks
	$Q bin/mockgen -source=internal/source/source.go -destination=internal/mocks/source.go -package=mocks
	$Q bin/mockgen -source=internal/mocks/api/client_stub.go -destination=internal/mocks/client.go -package=mocks
	$Q bin/mockgen -source=internal/domain/resolver.go -destination=internal/mocks/resolver.go -package=mocks

build: .GOPATH/.ok
	$Q GOBIN=$(CURDIR)/bin go install $(if $V,-v) $(VERSION_FLAGS) -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)

clean:
	$Q GOBIN=$(CURDIR)/bin go clean -i -modcache -x

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .
