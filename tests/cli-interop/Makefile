# Makefile for CLI Interop Tests
#
# This Makefile provides convenient targets for working specifically with the
# CLI interop test subsystem. For full project orchestration (including Go tests),
# use the root Makefile at ../../Makefile
#
# Quick reference:
#   Root Makefile: make test           - Run ALL tests (Go + .NET interop + CLI interop)
#   This Makefile: make test           - Run ONLY CLI interop tests
#   This Makefile: make test-verbose   - Run CLI interop tests with detailed output

.PHONY: help
help:
	@echo "CLI Interop Test Targets:"
	@echo "  make build      - Build gonuget CLI and CLI interop test bridge"
	@echo "  make restore    - Restore C# project dependencies"
	@echo "  make test       - Run CLI interop tests"
	@echo "  make clean      - Clean build artifacts"

.PHONY: build
build:
	@echo "Building gonuget CLI..."
	@cd ../../cmd/gonuget && go build -o ../../gonuget
	@echo "✓ Built gonuget"
	@echo "Building gonuget CLI interop test bridge..."
	@cd ../../cmd/gonuget-cli-interop-test && go build -o ../../gonuget-cli-interop-test
	@echo "✓ Built gonuget-cli-interop-test"

.PHONY: restore
restore:
	@echo "Restoring C# dependencies..."
	@cd GonugetCliInterop.Tests && dotnet restore
	@echo "✓ Dependencies restored"

.PHONY: test
test: build
	@echo "Running CLI interop tests..."
	@cd GonugetCliInterop.Tests && dotnet test --verbosity normal

.PHONY: test-verbose
test-verbose: build
	@echo "Running CLI interop tests (verbose)..."
	@cd GonugetCliInterop.Tests && dotnet test --verbosity detailed

.PHONY: clean
clean:
	@echo "Cleaning CLI interop test artifacts..."
	@rm -f ../../gonuget
	@rm -f ../../gonuget-cli-interop-test
	@cd GonugetCliInterop.Tests && dotnet clean
	@echo "✓ Cleaned"
