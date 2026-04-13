# Makefile for Flinn

.DEFAULT_GOAL := help

# Module directories (each has its own go.mod).
# yaml is excluded until it contains Go source files.
MODULES := cmd/flinn pkg/flinn pkg/source/toml

# Workspace-relative package patterns used by golangci-lint.
PKGS := ./cmd/flinn/... ./pkg/flinn/... ./pkg/source/toml/...

.PHONY: help
help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: fmt
fmt: # Format Go code (gofmt + goimports) across all modules.
	@echo "==> Formatting code..."
	@set -e; for m in $(MODULES); do (cd $$m && goimports -w .); done

.PHONY: test
test: # Run all tests with verbose output across all modules.
	@echo "==> Running tests..."
	@set -e; for m in $(MODULES); do (cd $$m && go test -v ./...); done

.PHONY: lint
lint: # Run golangci-lint for comprehensive static analysis.
	@echo "==> Linting code with golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run $(PKGS); \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi

.PHONY: vulncheck
vulncheck: # Check for known vulnerabilities using govulncheck.
	@echo "==> Checking for vulnerabilities with govulncheck..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		set -e; for m in $(MODULES); do (cd $$m && govulncheck ./...); done; \
	else \
		echo "govulncheck not found. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

.PHONY: verify
verify: fmt lint test vulncheck # Run all quality checks (fmt, lint, test, vulncheck).
	@echo "==> All quality checks passed successfully."

.PHONY: build
build: # Build the CLI demo/example.
	@echo "==> Building CLI demo..."
	@(cd cmd/flinn && go build -o ../../bin/flinn .)
