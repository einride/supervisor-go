x_tools_cwd := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
stringer := $(x_tools_cwd)/bin/stringer
export PATH := $(dir $(stringer)):$(PATH)

$(stringer): $(x_tools_cwd)/go.mod
	@echo building stringer...
	@cd $(x_tools_cwd) && go build -o $@ golang.org/x/tools/cmd/stringer
	@cd $(x_tools_cwd) && go mod tidy
