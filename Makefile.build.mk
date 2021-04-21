GO_BUILD_TAGS   := continuous_profiler_stackdriver

.PHONY: all setup generate-mocks build clean

all: gitlab-pages

setup: .GOPATH/.ok
	mkdir -p bin/
	# Installing dev tools defined in tools.go
	@cat ./tools.go | \
		grep _ | \
		awk -F'"' '{print $$2}' | \
		GOBIN=$(CURDIR)/bin xargs -tI % go install %

generate-mocks: .GOPATH/.ok
	$Q bin/mockgen -source=internal/interface.go -destination=internal/mocks/mocks.go -package=mocks

build: .GOPATH/.ok
	$Q GOBIN=$(CURDIR)/bin go install $(if $V,-v) $(VERSION_FLAGS) -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)

clean:
	$Q GOBIN=$(CURDIR)/bin go clean -i -modcache -x

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .
