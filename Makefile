# all: all tasks required for a complete build
.PHONY: all
all: build circleci-config-validate

# clean: remove generated build files
.PHONY: clean
clean:
	rm -rf build

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
