# Makefile for NuGet.Client Interop Tests
#
# This Makefile provides convenient targets for working specifically with the
# interop test subsystem. For full project orchestration (including Go tests),
# use the root Makefile at ../../Makefile
#
# Quick reference:
#   Root Makefile: make test           - Run ALL tests (Go + .NET interop)
#   This Makefile: make test           - Run ONLY .NET interop tests
#   This Makefile: make test-verbose   - Run interop tests with detailed output

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
