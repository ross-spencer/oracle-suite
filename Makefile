PACKAGE ?= gofer
GOFILES := $(shell { git ls-files; git ls-files -o --exclude-standard; } | grep ".go$$")

OUT_DIR := workdir
COVER_FILE := $(OUT_DIR)/cover.out

GO := go

clean:
	rm -rf $(OUT_DIR)
.PHONY: clean

vendor:
	$(GO) mod vendor
.PHONY: vendor

test:
	$(GO) test ./...
.PHONY: test

bench:
	$(GO) test -bench=. ./...
.PHONY: bench

lint:
	golangci-lint run ./...
.PHONY: lint

cover:
	@mkdir -p $(dir $(COVER_FILE))
	$(GO) test -coverprofile=$(COVER_FILE) ./...
	go tool cover -func=$(COVER_FILE)
.PHONY: cover

add-license: $(GOFILES)
	for x in $^; do tmp=$$(cat LICENSE_HEADER; sed -n '/^package /,$$p' $$x); echo "$$tmp" > $$x; done
.PHONY: add-license

test-license: $(GOFILES)
	@grep -vlz "$$(tr '\n' . < LICENSE_HEADER)" $^ && exit 1 || exit 0
.PHONY: test-license

test-all: test lint test-license
.PHONY: test-all
