.PHONY: build build-go build-dotnet build-interop test test-go test-dotnet test-interop clean help

# Default target
.DEFAULT_GOAL := help

# Build all components
build: build-go build-interop build-dotnet ## Build all Go packages and .NET projects

# Build all Go packages
build-go: ## Build all Go packages
	@echo "Building Go packages..."
	@go build ./...

# Build the interop test binary
build-interop: ## Build the gonuget-interop-test binary
	@echo "Building gonuget-interop-test binary..."
	@cd cmd/nuget-interop-test && go build -o ../../tests/nuget-client-interop/gonuget-interop-test .
	@echo "Binary built: tests/nuget-client-interop/gonuget-interop-test"

# Build .NET test project
build-dotnet: build-interop ## Build .NET interop test project (depends on interop binary)
	@echo "Building .NET test project..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet build

# Run all tests
test: test-go test-interop ## Run all tests (Go + .NET interop)

# Run Go tests
test-go: ## Run Go unit tests
	@echo "Running Go tests..."
	@go test ./... -v

# Run .NET interop tests
test-interop: build-interop build-dotnet ## Run .NET interop tests
	@echo "Running .NET interop tests..."
	@cd tests/nuget-client-interop/GonugetInterop.Tests && dotnet test --logger "console;verbosity=normal"

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
	@echo "Removing gonuget-interop-test binary..."
	@rm -f tests/nuget-client-interop/gonuget-interop-test
	@rm -f gonuget-interop-test
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
lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

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
