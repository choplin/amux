# Amux Justfile - Build automation

# Default recipe - show available commands
default:
    @just --list

# === Setup & Dependencies ===

# Initialize go modules and download dependencies
init:
    go mod download
    go mod tidy

# Check if required tools are available
check-tools:
    #!/usr/bin/env bash
    echo "Checking required tools..."
    
    # Check for npm/npx
    if ! command -v npx &> /dev/null; then
        echo "❌ npx not found. Please install Node.js/npm"
        echo "   Visit: https://nodejs.org/"
        exit 1
    else
        echo "✅ npx found"
    fi
    
    # Check for markdownlint-cli2
    if ! npx --no-install markdownlint-cli2 --version &> /dev/null 2>&1; then
        echo "❌ markdownlint-cli2 not found"
        echo "   Install with: npm install -g markdownlint-cli2"
    else
        echo "✅ markdownlint-cli2 found"
    fi
    
    # Check for commitlint
    if ! npx --no-install commitlint --version &> /dev/null 2>&1; then
        echo "❌ commitlint not found"
        echo "   Install with: npm install -g @commitlint/cli @commitlint/config-conventional"
    else
        echo "✅ commitlint found"
    fi
    
    echo ""
    echo "All required tools are available!"

# === Build & Install ===

# Build the binary
build:
    #!/usr/bin/env bash
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    go build -ldflags "-X github.com/aki/amux/internal/cli/commands.Version=$VERSION -X github.com/aki/amux/internal/cli/commands.GitCommit=$COMMIT -X github.com/aki/amux/internal/cli/commands.BuildDate=$DATE" -o bin/amux cmd/amux/main.go

# Install the binary to GOPATH/bin
install: build
    go install cmd/amux/main.go

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f coverage.out coverage.html

# === Testing ===

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# === Formatting ===

# Format Go code
fmt-go:
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./...

# Format YAML files
fmt-yaml:
    go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt .

# Fix markdown files
fmt-md:
    npm run fix:md

# Fix trailing spaces and ensure newline at EOF
fmt-whitespace:
    #!/usr/bin/env bash
    # Remove trailing spaces
    find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "*.txt" -o -name "*.json" -o -name "*.toml" -o -name "*.mod" -o -name "*.sum" \) \
        -not -path "./vendor/*" -not -path "./.git/*" -not -path "./bin/*" \
        -exec perl -i -pe 's/[ \t]+$//' {} \;
    # Ensure newline at EOF
    find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "*.txt" -o -name "*.json" -o -name "*.toml" -o -name "*.mod" -o -name "*.sum" \) \
        -not -path "./vendor/*" -not -path "./.git/*" -not -path "./bin/*" \
        -exec perl -i -pe 'eof && do{print "\n" unless /\n$/}' {} \;

# Format all code
fmt: fmt-whitespace fmt-go fmt-yaml fmt-md

# === Linting ===

# Lint Go code
lint-go:
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

# Lint markdown files
lint-md:
    npm run lint:md

# Lint all code
lint: lint-go lint-md

# === Combined Commands ===

# Check code (format + lint) - matches pre-commit hooks
check: fmt lint

# Quick check without fixing (for CI)
check-ci: lint
    #!/usr/bin/env bash
    # Check for formatting changes
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./... --diff
    # Check for yaml formatting
    go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt -dry .

# Full development cycle - format, lint, test, build
all: check test build

# === Development Helpers ===

# Show current version
version:
    @git describe --tags --always --dirty 2>/dev/null || echo "dev"

# Run the development version
dev *args:
    go run cmd/amux/main.go {{args}}

# Create a new workspace (development helper)
workspace-create name:
    just dev workspace create {{name}}

# List workspaces (development helper)
workspace-list:
    just dev workspace list

# Start MCP server
serve:
    just dev serve

# Watch for changes and rebuild
watch:
    watchexec -e go -r "just build"