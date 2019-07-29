# all: all tasks required for a complete build
.PHONY: all
all: circleci-config-validate \
	go-generate \
	go-lint \
	go-review \
	go-test \
	go-mod-tidy \
	git-verify-nodiff

# clean: remove generated build files
.PHONY: clean
clean:
	rm -rf build

export GO111MODULE = on

.PHONY: build
build:
	@git submodule update --init --recursive $@

include build/rules.mk
build/rules.mk: build
	@# included in submodule: build

# circleci-config-validate: validate CircleCI config
.PHONY: circleci-config-validate
circleci-validate-config: $(CIRCLECI)
	$(CIRCLECI) config validate

# go-mod-tidy: make sure go module is neat and tidy
.PHONY: go-mod-tidy
go-mod-tidy:
	go mod tidy

# go-lint: lint Go code with GolangCI-Lint
go-lint: $(GOLANGCI_LINT) gofumports-verify-format
	$(GOLANGCI_LINT) run --enable-all --skip-dirs build

# go-generate: generate Go code using `go generate`
.PHONY: go-generate
go-generate: status_string.go

status_string.go: status.go
	go generate $<

# go-review: review Go code with goreview
.PHONY: go-review
go-review: $(GOREVIEW)
	$(GOREVIEW) -c 1 ./...

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -count=1 -race -cover ./...
