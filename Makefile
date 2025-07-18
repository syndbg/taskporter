.PHONY: build test lint clean install dev help

# Build binary
build:
	@echo "Building taskporter..."
	@go build -o bin/taskporter .

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@GOOS=linux GOARCH=amd64 go build -o bin/taskporter-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -o bin/taskporter-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -o bin/taskporter-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build -o bin/taskporter-windows-amd64.exe .

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Development setup
dev: install
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the application
run:
	@go run . $(ARGS)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install dependencies"
	@echo "  dev          - Setup development environment"
	@echo "  run          - Run the application (use ARGS=... for arguments)"
	@echo "  fmt          - Format code"
	@echo "  help         - Show this help"
