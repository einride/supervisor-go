# all: all tasks required for a complete build
.PHONY: all
all: circleci-config-validate \
	go-lint \
	go-review \
	go-test \
	go-mod-tidy \

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
go-lint: $(GOLANGCI_LINT) $(GOFUMPORTS)
	$(GOLANGCI_LINT) run --enable-all --skip-dirs build
	@set -e; \
	go_files=$$(git ls-files --exclude-standard --cached --others '*.go'); \
	not_formatted=$$($(GOFUMPORTS) -l $$go_files); \
	if ! test -z "$$not_formatted"; then \
		echo 'Files not `gofumports`-ed:'; \
		echo "$$not_formatted"; \
		exit 1; \
	fi

# go-review: review Go code with goreview
.PHONY: go-review
go-review: $(GOREVIEW)
	$(GOREVIEW) -c 1 ./...

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -count=1 -race -cover ./...
