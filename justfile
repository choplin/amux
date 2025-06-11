# Amux Justfile - Build automation

# Default recipe - show available commands
default:
    @just --list

# Initialize go modules and download dependencies
init:
    go mod download
    go mod tidy

# Build the binary
build:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    go build -ldflags "-X github.com/aki/amux/internal/cli/commands.Version=$VERSION -X github.com/aki/amux/internal/cli/commands.GitCommit=$COMMIT -X github.com/aki/amux/internal/cli/commands.BuildDate=$DATE" -o bin/amux cmd/amux/main.go

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Format Go code
fmt:
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./...

# Format YAML files
fmt-yaml:
    go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt .

# Lint code
lint:
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

# Check code (format + lint)
check: fmt fmt-yaml lint

# Install the binary to GOPATH/bin
install: build
    go install cmd/amux/main.go

# Show current version
version:
    @git describe --tags --always --dirty 2>/dev/null || echo "dev"

# Run the development version
dev *args:
    go run cmd/amux/main.go {{args}}

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f coverage.out coverage.html

# Create a new workspace (development helper)
workspace name:
    just dev workspace create {{name}}

# List workspaces (development helper)
list:
    just dev workspace list

# Start MCP server
serve:
    just dev serve

# Full development cycle - format, lint, test, build
all: fmt fmt-yaml lint test build

# Watch for changes and rebuild
watch:
    watchexec -e go -r "just build"