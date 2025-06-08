# AgentCave Makefile

# Variables
BINARY_NAME=agentcave
BINARY_DIR=bin
GO_FILES=$(shell find . -name '*.go' -type f)
MAIN_PACKAGE=./cmd/agentcave

# Default target
.PHONY: all
all: build

# Initialize go modules
.PHONY: init
init:
	go mod download
	go mod tidy

# Build the binary
.PHONY: build
build: $(BINARY_DIR)/$(BINARY_NAME)

$(BINARY_DIR)/$(BINARY_NAME): $(GO_FILES)
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# Run tests
.PHONY: test
test:
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
.PHONY: fmt
fmt:
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not installed, skipping imports formatting"; \
	fi

# Lint code
.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping linting"; \
	fi

# Check code (format + lint)
.PHONY: check
check: fmt lint

# Install the binary
.PHONY: install
install: build
	go install $(MAIN_PACKAGE)

# Run the development version
.PHONY: dev
dev:
	go run $(MAIN_PACKAGE) $(ARGS)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Create a new workspace (development helper)
.PHONY: workspace
workspace:
	go run $(MAIN_PACKAGE) workspace create $(NAME)

# List workspaces (development helper)
.PHONY: list
list:
	go run $(MAIN_PACKAGE) workspace list

# Start MCP server
.PHONY: serve
serve:
	go run $(MAIN_PACKAGE) serve

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build the binary (default)"
	@echo "  init         - Initialize go modules"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  check        - Format and lint code"
	@echo "  install      - Install the binary"
	@echo "  dev          - Run the development version"
	@echo "  clean        - Clean build artifacts"
	@echo "  workspace    - Create a new workspace (NAME=name)"
	@echo "  list         - List workspaces"
	@echo "  serve        - Start MCP server"
	@echo "  help         - Show this help message"