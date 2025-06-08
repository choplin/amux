# AgentCave Justfile - Build automation

# Default recipe - show available commands
default:
    @just --list

# Initialize go modules and download dependencies
init:
    go mod download
    go mod tidy

# Build the binary
build:
    go build -o bin/agentcave cmd/agentcave/main.go

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Format Go code
fmt:
    go run -mod=readonly golang.org/x/tools/cmd/goimports -w -local github.com/aki/agentcave .

# Format YAML files
fmt-yaml:
    go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt .

# Lint code
lint:
    go run -mod=readonly github.com/golangci/golangci-lint/cmd/golangci-lint run

# Check code (format + lint)
check: fmt fmt-yaml lint

# Install the binary to GOPATH/bin
install: build
    go install cmd/agentcave/main.go

# Run the development version
dev *args:
    go run cmd/agentcave/main.go {{args}}

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