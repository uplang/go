# Makefile for UP Go Parser

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: lint
lint: ## Run linter
	go tool golangci-lint run ./...

.PHONY: build
build: ## Build the library
	go build -v ./...

.PHONY: clean
clean: ## Clean build artifacts
	go clean
	rm -f coverage.out

.PHONY: install
install: ## Install dependencies
	go mod download
	go mod tidy

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test-ci
test-ci: ## Run CI tests locally using act (requires: brew install act)
	act --container-architecture linux/amd64 -j test
	act --container-architecture linux/amd64 -j lint
	act --container-architecture linux/amd64 -j build

.DEFAULT_GOAL := test
