GOCACHE := $(CURDIR)/.cache/go-build
GOLANGCI_LINT_CACHE := $(CURDIR)/.cache/golangci-lint
BINARY := bin/sway-compat

export GOCACHE
export GOLANGCI_LINT_CACHE

.PHONY: help build run test test-coverage setup-ci fmt lint install install-local release-snapshot clean tail-log tail-log-truncate

help: ## Lists the available commands. Add '##' to describe a command.
	@grep -E '^[a-zA-Z_-].+:.*?## .*$$' $(MAKEFILE_LIST)\
		| sort\
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

build: ## Build the CLI
	@echo "Building sway-compat..."
	@mkdir -p $(dir $(BINARY))
	@go build -o $(BINARY) main.go

run: ## Run the CLI
	@echo "Running sway-compat..."
	@go run main.go

test: ## Run tests
	@echo "Running tests..."
	@go test ./... -v

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

setup-ci: ## Install dependencies for CI
	@echo "Setting up CI dependencies..."
	@mkdir -p $(GOCACHE) $(GOLANGCI_LINT_CACHE)
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0; \
	else \
		echo "golangci-lint already installed"; \
	fi

fmt: setup-ci ## Format the code
	@echo "Formatting code..."
	@gofmt -s -w .
	@golangci-lint run --fix

lint: setup-ci ## Run the linter
	@echo "Running linter..."
	@golangci-lint run

install: build ## Install the CLI system-wide to /usr/local/bin
	@echo "Installing sway-compat to /usr/local/bin..."
	@install -Dm755 $(BINARY) /usr/local/bin/sway-compat

install-local: build ## Install the CLI for the current user
	@echo "Installing sway-compat to ~/.local/bin..."
	@install -Dm755 $(BINARY) $(HOME)/.local/bin/sway-compat

release-snapshot: ## Build local release artifacts without publishing
	@go run github.com/goreleaser/goreleaser/v2@v2.13.3 release --snapshot --clean

clean: ## Clean build artifacts
	@echo "Cleaning artifacts..."
	@rm -rf $(BINARY) coverage.out coverage.html dist

tail-log: ## Tail the log file
	@echo "Tailing /tmp/sway-compat.log..."
	@tail -f /tmp/sway-compat.log

tail-log-truncate: ## Tail the log file and truncate it when it exceeds 10MB
	@echo "Truncating and tailing /tmp/sway-compat.log..."
	@truncate -s 0 /tmp/sway-compat.log && tail -f /tmp/sway-compat.log
