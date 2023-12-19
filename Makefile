.DEFAULT_GOAL := help

.PHONY: lint snapshot help

lint:                                   ## Lint the source code (--ignore-errors to ignore errs)
	staticcheck ./...
	go fmt ./...
	golint ./...
	go vet ./...

pre-commit-checks:                      ## Run pre-commit-checks.
	pre-commit run --all-files

help:                                   ## Print this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'