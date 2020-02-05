.PHONY: all setup generate-mocks build clean

all: gitlab-pages

setup: clean .GOPATH/.ok
	go get golang.org/x/tools/cmd/goimports@v0.0.0-20191010201905-e5ffc44a6fee
	go get golang.org/x/lint/golint@v0.0.0-20190930215403-16217165b5de
	go get github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	go get github.com/fzipp/gocyclo@v0.0.0-20150627053110-6acd4345c835
	go get github.com/golang/mock/mockgen@v1.3.1

generate-mocks: .GOPATH/.ok
	$Q bin/mockgen -source=internal/interface.go -destination=internal/mocks/mocks.go -package=mocks

build: .GOPATH/.ok
	$Q go install $(if $V,-v) $(VERSION_FLAGS) -buildmode exe $(IMPORT_PATH)

clean:
	$Q rm -rf bin .GOPATH gitlab-pages

gitlab-pages: build
	$Q cp -f ./bin/gitlab-pages .

