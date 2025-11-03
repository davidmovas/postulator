.PHONY: test test-verbose test-coverage clean help

# Default target
help:
	@echo "Available commands:"
	@echo "  make test           - Run all Go tests"
	@echo "  make test-verbose   - Run all Go tests with verbose output"
	@echo "  make test-coverage  - Run all Go tests with coverage report"
	@echo "  make clean          - Clean test cache and temporary files"

# Run all tests
test:
	@echo "Running all Go tests..."
	@go test ./... -timeout 30s

# Run all tests with verbose output
test-verbose:
	@echo "Running all Go tests (verbose)..."
	@go test -v ./... -timeout 30s

# Run all tests with coverage
test-coverage:
	@echo "Running all Go tests with coverage..."
	@go test -v -cover -coverprofile=coverage.out ./... -timeout 30s
	@echo ""
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | grep total

# Clean test cache and temporary files
clean:
	@echo "Cleaning test cache and temporary files..."
	@go clean -testcache
	@if exist coverage.out del /F /Q coverage.out
	@echo "Done."

go-lint:
	golangci-lint run ./...

go-fmt:
	 gofumpt -l -w .
