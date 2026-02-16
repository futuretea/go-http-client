.PHONY: help build test lint fmt vet clean

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the package
	go build ./...

test: ## Run all tests
	go test -v ./...

test-race: ## Run tests with race detector
	go test -race -v ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt: ## Format code with gofmt and goimports
	gofmt -w .
	@which goimports > /dev/null 2>&1 && goimports -w . || echo "goimports not installed, skipping"

vet: ## Run go vet
	go vet ./...

check: fmt vet test ## Run fmt, vet and test

clean: ## Clean build artifacts
	go clean -cache
	go clean -testcache
