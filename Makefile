.PHONY: build build-go build-dotnet build-interop test test-go test-go-unit test-dotnet test-interop clean help

# Default target
.DEFAULT_GOAL := help

# Build all components
build: build-go build-interop build-cli build-cli-interop build-dotnet ## Build all Go packages, CLI, and .NET projects

# Build all Go packages
build-go: ## Build all Go packages
	@echo "Building Go packages..."
	@go build ./...

# Build the interop test binary
build-interop: ## Build the gonuget-interop-test binary
	@echo "Building gonuget-interop-test binary..."
	@cd cmd/nuget-interop-test && go build -o ../../gonuget-interop-test .
	@echo "Binary built: gonuget-interop-test"

# Build the CLI interop test binary
build-cli-interop: ## Build the gonuget-cli-interop-test binary
	@echo "Building gonuget-cli-interop-test binary..."
	@cd cmd/gonuget-cli-interop-test && go build -o ../../gonuget-cli-interop-test .
	@echo "Binary built: gonuget-cli-interop-test"

# Build the gonuget CLI
build-cli: ## Build the gonuget CLI executable
	@echo "Building gonuget CLI..."
	@go build -o gonuget ./cmd/gonuget
	@echo "Binary built: gonuget"

# Build .NET test project
build-dotnet: build-interop ## Build .NET interop test project (depends on interop binary)
	@echo "Building .NET test project..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet build

# Run all tests
test: test-go test-interop test-cli-interop ## Run all tests (Go unit + integration + .NET interop + CLI interop)

# Run Go tests (unit + integration)
test-go: ## Run all Go tests (unit + integration tests that hit nuget.org)
	@echo "Running Go tests (unit + integration)..."
	@go test ./... -v

# Run Go unit tests only (skip integration tests)
test-go-unit: ## Run only Go unit tests (skip integration tests that hit nuget.org)
	@echo "Running Go unit tests only (skipping integration)..."
	@go test ./... -v -short

# Run .NET interop tests
test-interop: build-interop build-dotnet ## Run .NET interop tests
	@echo "Running .NET interop tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --logger "console;verbosity=normal"

# Run CLI interop tests
test-cli-interop: build-cli build-cli-interop ## Run CLI interop tests (dotnet nuget vs gonuget)
	@echo "Running CLI interop tests..."
	@cd tests/cli-interop/GonugetCliInterop.Tests && dotnet test --logger "console;verbosity=normal"

# Run all interop tests (library + CLI)
test-all-interop: test-interop test-cli-interop ## Run all interop tests (library + CLI)

# Run only version tests
test-version: build-interop build-dotnet ## Run only version interop tests
	@echo "Running version interop tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --filter "FullyQualifiedName~VersionTests" --logger "console;verbosity=normal"

# Run only smoke tests
test-smoke: build-interop build-dotnet ## Run only smoke tests
	@echo "Running smoke tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --filter "FullyQualifiedName~BridgeSmokeTests" --logger "console;verbosity=normal"

# Run only signature tests
test-signature: build-interop build-dotnet ## Run only signature tests
	@echo "Running signature tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --filter "FullyQualifiedName~SignatureTests" --logger "console;verbosity=normal"

# Quick rebuild and test (for development)
quick-test: build-interop ## Quick rebuild interop binary and run tests (no clean)
	@echo "Quick test: rebuilding and running tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --logger "console;verbosity=normal"

# Clean build artifacts
clean: ## Clean all build artifacts
	@echo "Cleaning Go build cache..."
	@go clean -cache
	@echo "Cleaning .NET build artifacts..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet clean
	@if [ -d tests/cli-interop/GonugetCliInterop.Tests ]; then \
		cd tests/cli-interop/GonugetCliInterop.Tests && dotnet clean; \
	fi
	@echo "Removing gonuget-interop-test binary..."
	@rm -f gonuget-interop-test
	@echo "Removing gonuget-cli-interop-test binary..."
	@rm -f gonuget-cli-interop-test
	@echo "Removing gonuget CLI binary..."
	@rm -f gonuget
	@echo "Clean complete"

# Show test count
test-count: build-dotnet ## Show test count by category
	@echo "Test count by category:"
	@echo "======================"
	@cd tests/nuget-client-interop/GonugetInterop.Tests && \
		dotnet test --list-tests | grep -E "(Smoke|Signature|Version|Framework|Package)" | \
		awk -F'.' '{print $$NF}' | awk -F'(' '{print $$1}' | sort | uniq -c | \
		awk '{printf "%-30s %3d tests\n", $$2, $$1}'

# Format Go code
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@gofmt -w .

# Run linter
lint: ## Run golangci-lint and modernize checks
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "Running modernize checks..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest ./...

# Help target
help: ## Show this help message
	@echo "gonuget Makefile"
	@echo "================"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
