.PHONY: build test lint coverage pre-commit install-tools clean help

# Build configuration
BINARY_NAME := confvis
BUILD_DIR := ./cmd/confvis
COVERAGE_FILE := coverage.out
COVERAGE_THRESHOLD := 80

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o $(BINARY_NAME) $(BUILD_DIR)

test: ## Run all tests
	go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...

lint: ## Run golangci-lint
	golangci-lint run --config .golangci.yml ./...

coverage: test ## Run tests and check coverage threshold
	@go tool cover -func=$(COVERAGE_FILE) | grep total
	@coverage=$$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{gsub(/%/,""); print int($$3)}'); \
	if [ $$coverage -lt $(COVERAGE_THRESHOLD) ]; then \
		echo "Coverage $$coverage% is below threshold of $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "Coverage $$coverage% meets threshold of $(COVERAGE_THRESHOLD)%"; \
	fi

coverage-html: test ## Generate HTML coverage report
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report written to coverage.html"

pre-commit: lint test coverage ## Run all pre-commit checks (lint, test, coverage)
	@echo "All pre-commit checks passed"

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installed golangci-lint"
	@if command -v npm > /dev/null; then \
		npm install -g @commitlint/cli @commitlint/config-conventional; \
		echo "Installed commitlint"; \
	else \
		echo "npm not found, skipping commitlint installation"; \
	fi
	@if command -v pre-commit > /dev/null; then \
		pre-commit install; \
		echo "Installed pre-commit hooks"; \
	else \
		echo "pre-commit not found, skipping hook installation"; \
	fi

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(COVERAGE_FILE) coverage.html
	go clean -testcache

# Local development helpers
run: build ## Build and run with example args
	./$(BINARY_NAME) --help

test-verbose: ## Run tests with verbose output
	go test -race -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
