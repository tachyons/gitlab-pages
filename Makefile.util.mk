.PHONY: lint test race acceptance bench cover list deps-check deps-download changelog

OUT_FORMAT ?= colored-line-number
LINT_FLAGS ?=  $(if $V,-v)
REPORT_FILE ?=

lint: .GOPATH/.ok bin/golangci-lint
	$Q ./bin/golangci-lint run ./... --out-format $(OUT_FORMAT) $(LINT_FLAGS) | tee ${REPORT_FILE}

test: .GOPATH/.ok gitlab-pages
	rm -f tests.out
	go test $(if $V,-v) ./... ${ARGS} 2>&1 | tee tests.out

race: .GOPATH/.ok gitlab-pages
	CGO_ENABLED=1 go test -race $(if $V,-v) $(allpackages) 2>&1 | tee tests.out

acceptance: .GOPATH/.ok gitlab-pages
	go test $(if $V,-v) ./test/acceptance ${ARGS} 2>&1 | tee tests.out

bench: .GOPATH/.ok gitlab-pages
	go test -bench=. -run=^$$ $(allpackages)

# The acceptance tests cannot count for coverage
cover: bin/gocovmerge gitlab-pages
	@echo "NOTE: make cover does not exit 1 on failure, don't use it to check for tests success!"
	$Q rm -f .GOPATH/cover/*.out .GOPATH/cover/all.merged
	$Q mkdir -p .GOPATH/cover
	$(if $V,@echo "-- go test -coverpkg=./... -coverprofile=.GOPATH/cover/... ./...")
	@for MOD in $(allpackages); do \
		go test \
			-short \
			-coverpkg=`echo $(allpackages)|tr " " ","` \
			-coverprofile=.GOPATH/cover/unit-`echo $$MOD|tr "/" "_"`.out \
			$$MOD 2>&1 | grep -v "no packages being tested depend on"; \
	done
	$Q ./bin/gocovmerge .GOPATH/cover/*.out > .GOPATH/cover/all.merged
	$Q go tool cover -html .GOPATH/cover/all.merged -o coverage.html
	@echo ""
	@echo "=====> Total test coverage: <====="
	@echo ""
	$Q go tool cover -func .GOPATH/cover/all.merged

list: .GOPATH/.ok
	@echo $(allpackages)

deps-check: .GOPATH/.ok
	go mod tidy
	@if git diff --color=always --exit-code -- go.mod go.sum; then \
		echo "go.mod and go.sum are ok"; \
	else \
    echo ""; \
		echo "go.mod and go.sum are modified, please commit them";\
		exit 1; \
  fi;

deps-download: .GOPATH/.ok
	go mod download

junit-report: .GOPATH/.ok bin/go-junit-report
	cat tests.out | ./bin/go-junit-report -set-exit-code > junit-test-report.xml

changelog:
	TOKEN='$(GITLAB_PRIVATE_TOKEN)' VERSION='$(shell cat VERSION)' BRANCH='$(BRANCH)'  bash ./.gitlab/scripts/changelog.sh
ifndef GITLAB_PRIVATE_TOKEN
	$(error GITLAB_PRIVATE_TOKEN is undefined)
endif
