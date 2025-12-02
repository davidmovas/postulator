.PHONY: test test-verbose test-coverage clean help commit

# Default target
help:
	@echo "Available commands:"
	@echo "  make test           - Run all Go tests"
	@echo "  make test-verbose   - Run all Go tests with verbose output"
	@echo "  make test-coverage  - Run all Go tests with coverage report"
	@echo "  make clean          - Clean test cache and temporary files"
	@echo "  make commit         - Create conventional commit using Claude AI"

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

# Create a conventional commit using Claude AI
commit:
	@npx @anthropic-ai/claude-code -p "Analyze git status and git diff, then create a conventional commit for all current changes. Use conventional commit format (feat:, fix:, docs:, chore:, refactor:, etc). Write commit message in English. Stage all changes with 'git add -A' first, then commit. After committing, show the commit hash and message. Do not push."