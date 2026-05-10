#!/bin/bash
set -e

echo "Setting up HybridDB development environment..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.22+."
    exit 1
fi

# Check for golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.0
fi

# Download dependencies
go mod tidy

echo "Setup complete! Run 'make build' to verify."
