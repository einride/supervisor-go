# all: all tasks required for a complete build
.PHONY: all
all: \
	prettier-yaml \
	prettier-markdown \
	go-generate \
	go-lint \
	go-review \
	go-test \
	go-mod-tidy \
	git-verify-nodiff

include tools/git-verify-nodiff/rules.mk
include tools/golangci-lint/rules.mk
include tools/prettier/rules.mk
include tools/goreview/rules.mk
include tools/xtools/rules.mk
include tools/semantic-release/rules.mk

# prettier-yaml: formats all yaml files with prettier
.PHONY: prettier-yaml
prettier-yaml: $(prettier)
	$(prettier) --check ./**/*.y*ml --check *.y*ml --parser yaml --write

# prettier-markdown: formats all Markdown files with prettier
.PHONY: prettier-markdown
prettier-markdown: $(prettier)
	$(prettier) --check **/*.md --parser markdown --write

# go-mod-tidy: make sure go module is neat and tidy
.PHONY: go-mod-tidy
go-mod-tidy:
	find . -name go.mod -execdir go mod tidy \;

# go-generate: generate Go code using `go generate`
.PHONY: go-generate
go-generate: status_string.go

status_string.go: status.go $(stringer)
	$(stringer) -type Status -trimprefix Status $<

# go-test: run Go test suite
.PHONY: go-test
go-test:
	go test -count=1 -race -cover ./...
