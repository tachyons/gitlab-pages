.PHONY: all setup generate-mocks build clean

all: gitlab-pages

setup: clean .GOPATH/.ok
	go get golang.org/x/tools/cmd/goimports
	go get golang.org/x/lint/golint
	go get github.com/wadey/gocovmerge
	go get github.com/fzipp/gocyclo
	go get github.com/golang/mock/mockgen

generate-mocks: .GOPATH/.ok
	$Q bin/mockgen -source=internal/interface.go -destination=internal/mocks/mocks.go -package=mocks

build: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) -buildmode exe $(IMPORT_PATH)

clean:
	$Q rm -rf bin .GOPATH gitlab-pages

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .

