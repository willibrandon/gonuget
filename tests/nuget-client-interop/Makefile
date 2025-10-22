# Makefile for NuGet.Client Interop Tests

.PHONY: help
help:
	@echo "NuGet.Client Interop Test Targets:"
	@echo "  make build      - Build gonuget interop CLI"
	@echo "  make restore    - Restore C# project dependencies"
	@echo "  make test       - Run interop tests"
	@echo "  make clean      - Clean build artifacts"

.PHONY: build
build:
	@echo "Building gonuget interop CLI..."
	@cd ../../cmd/nuget-interop-test && go build -o ../../gonuget-interop-test
	@echo "✓ Built gonuget-interop-test"

.PHONY: restore
restore:
	@echo "Restoring C# dependencies..."
	@cd GonugetInterop.Tests && dotnet restore
	@echo "✓ Dependencies restored"

.PHONY: test
test: build
	@echo "Running NuGet.Client interop tests..."
	@cd GonugetInterop.Tests && dotnet test --verbosity normal

.PHONY: test-verbose
test-verbose: build
	@echo "Running NuGet.Client interop tests (verbose)..."
	@cd GonugetInterop.Tests && dotnet test --verbosity detailed

.PHONY: clean
clean:
	@echo "Cleaning interop test artifacts..."
	@rm -f ../../gonuget-interop-test
	@cd GonugetInterop.Tests && dotnet clean
	@echo "✓ Cleaned"
