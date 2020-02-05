.PHONY: verify fmt vet lint complexity test cover list

verify: list fmt vet lint complexity

fmt: bin/goimports .GOPATH/.ok
	$Q @_support/validate-formatting.sh $(allfiles)

vet: .GOPATH/.ok
	$Q go vet $(allpackages)

lint: bin/golint
	$Q ./bin/golint $(allpackages) | tee | ( ! grep -v "^$$" )

complexity: .GOPATH/.ok bin/gocyclo
	$Q ./bin/gocyclo -over 9 $(allfiles)

test: .GOPATH/.ok gitlab-pages
	go test $(if $V,-v) $(allpackages)

race: .GOPATH/.ok gitlab-pages
	CGO_ENABLED=1 go test -race $(if $V,-v) $(allpackages)

acceptance: .GOPATH/.ok gitlab-pages
	go test $(if $V,-v) $(IMPORT_PATH)

bench: .GOPATH/.ok gitlab-pages
	go test -bench=. -run=^$$ $(allpackages)

benchstat: .GOPATH/.ok bin/benchstat
	@_support/benchmark

# The acceptance tests cannot count for coverage
cover: bin/gocovmerge .GOPATH/.ok gitlab-pages
	@echo "NOTE: make cover does not exit 1 on failure, don't use it to check for tests success!"
	$Q rm -f .GOPATH/cover/*.out .GOPATH/cover/all.merged
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
