GOLANGCI_LINT_VERSION := v1.27.0 # version used by $GOLANGCI_LINT_IMAGE

GO_BUILD_TAGS   := continuous_profiler_stackdriver

.PHONY: all setup generate-mocks build clean

all: gitlab-pages

setup: clean .GOPATH/.ok
	go get github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	go get github.com/golang/mock/mockgen@v1.3.1
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

generate-mocks: .GOPATH/.ok
	$Q bin/mockgen -source=internal/interface.go -destination=internal/mocks/mocks.go -package=mocks

build: .GOPATH/.ok
	$Q GOBIN=$(CURDIR)/bin go install $(if $V,-v) $(VERSION_FLAGS) -tags "${GO_BUILD_TAGS}" -buildmode exe $(IMPORT_PATH)

clean:
	$Q rm -rf bin .GOPATH gitlab-pages

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .

