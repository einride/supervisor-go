prettier_version := 1.19.1
prettier_cwd := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/v$(prettier_version)
prettier := $(prettier_cwd)/node_modules/.bin/prettier

$(prettier):
	$(info installing prettier...)
	@npm install --no-save --no-audit --prefix $(prettier_cwd) prettier@$(prettier_version)
	@chmod +x $@
	@touch $@
