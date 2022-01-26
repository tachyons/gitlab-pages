.PHONY: lint test race acceptance bench cover list deps-check deps-download changelog

OUT_FORMAT ?= colored-line-number
LINT_FLAGS ?=  $(if $V,-v)
REPORT_FILE ?=
COVERAGE_PACKAGES=$(shell (go list ./... | grep -v $(addprefix -e ,$(IGNORED_DIRS) "test/acceptance") | tr "\n", "," | sed 's/\(.*\),/\1 /'))

lint: .GOPATH/.ok bin/golangci-lint
	$Q ./bin/golangci-lint run ./... --out-format $(OUT_FORMAT) $(LINT_FLAGS) | tee ${REPORT_FILE}

format: .GOPATH/.ok bin/golangci-lint
	$Q ./bin/golangci-lint run ./... --fix --out-format $(OUT_FORMAT) $(LINT_FLAGS) | tee ${REPORT_FILE}

test: .GOPATH/.ok .GOPATH/.ok bin/gotestsum gitlab-pages
	rm -f tests.out
	./bin/gotestsum --junitfile junit-test-report.xml --format testname -- ./... ${ARGS}

race: .GOPATH/.ok .GOPATH/.ok bin/gotestsum gitlab-pages
	CGO_ENABLED=1 ./bin/gotestsum --junitfile junit-test-report.xml --format testname -- -race $(if $V,-v) $(allpackages) ${ARGS}

acceptance: .GOPATH/.ok .GOPATH/.ok bin/gotestsum gitlab-pages
	./bin/gotestsum --junitfile junit-test-report.xml --format testname -- $(if $V,-v) ./test/acceptance ${ARGS}

bench: .GOPATH/.ok gitlab-pages
	go test -bench=. -run=^$$ $(allpackages)

# The acceptance tests cannot count for coverage
cover: gitlab-pages
	@echo "NOTE: make cover does not exit 1 on failure, don't use it to check for tests success!"
	$Q rm -f .GOPATH/cover/test.coverage
	$Q mkdir -p .GOPATH/cover
	$Q go test -short -cover -coverpkg=$(COVERAGE_PACKAGES) -coverprofile=.GOPATH/cover/test.coverage $(allpackages)
	$Q go tool cover -html .GOPATH/cover/test.coverage -o coverage.html
	@echo ""
	@echo "=====> Total test coverage: <====="
	@echo ""
	$Q go tool cover -func .GOPATH/cover/test.coverage

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

mocks-check: .GOPATH/.ok generate-mocks
	@if git diff --color=always --exit-code -- *_mock.go; then \
		echo "mocks are ok"; \
	else \
    echo ""; \
		echo "mocks needs to be regenerated, please run 'make generate-mocks' and commit them";\
		exit 1; \
  fi;

deps-download: .GOPATH/.ok
	go mod download

junit-report: .GOPATH/.ok bin/gotestsum
	./bin/gotestsum --junitfile junit-test-report.xml --format testname

changelog:
	TOKEN='$(GITLAB_PRIVATE_TOKEN)' VERSION='$(shell cat VERSION)' BRANCH='$(BRANCH)'  bash ./.gitlab/scripts/changelog.sh
ifndef GITLAB_PRIVATE_TOKEN
	$(error GITLAB_PRIVATE_TOKEN is undefined)
endif

.PHONY: zip

zip:
	cd $(PWD)/shared/pages/$(PROJECT_SUBDIR)/ && \
	zip -r public.zip public/
