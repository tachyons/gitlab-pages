.PHONY: all setup build clean

all: gitlab-pages

setup: clean .GOPATH/.ok
	go get -u github.com/FiloSottile/gvt
	- ./bin/gvt fetch golang.org/x/tools/cmd/goimports
	- ./bin/gvt fetch github.com/wadey/gocovmerge
	- ./bin/gvt fetch github.com/golang/lint/golint
	- ./bin/gvt fetch github.com/fzipp/gocyclo

build: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) $(IMPORT_PATH)

clean:
	$Q rm -rf bin .GOPATH gitlab-pages

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .

