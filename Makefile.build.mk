.PHONY: all setup build clean

all: gitlab-pages

setup: clean .GOPATH/.ok
	go get golang.org/x/tools/cmd/goimports
	go get golang.org/x/lint/golint
	go get github.com/wadey/gocovmerge
	go get github.com/fzipp/gocyclo

build: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) -buildmode exe $(IMPORT_PATH)

clean:
	$Q rm -rf bin .GOPATH gitlab-pages

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .

