# Amux Justfile - Build automation
#
# This justfile provides a single source of truth for linting and formatting commands.
# All formatting/linting tasks accept optional file arguments:
#   - No args: process all files (development workflow)
#   - With args: process specific files (used by git hooks)
#
# Examples:
#   just fmt-go                    # Format all Go files
#   just fmt-go file1.go file2.go  # Format specific files
#   just lint                      # Run all linters
#   just check                     # Format and lint everything

# Default recipe - show available commands
default:
    @just --list

# === Setup & Dependencies ===

# Initialize project dependencies
init:
    #!/usr/bin/env bash
    echo "📦 Initializing Go modules..."
    go mod download
    go mod tidy

    echo "📦 Installing Go tools..."
    # Download golangci-lint
    go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint version > /dev/null 2>&1
    echo "✅ golangci-lint ready"

    # Download yamlfmt
    go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt -version > /dev/null 2>&1
    echo "✅ yamlfmt ready"

    echo "📦 Installing npm dependencies..."
    if command -v npm &> /dev/null; then
        npm install
        echo "✅ npm packages installed"
    else
        echo "❗ npm not found. Install Node.js to use markdown linting"
        echo "   Visit: https://nodejs.org/"
    fi

    echo ""
    echo "✅ All dependencies initialized!"

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

# Run short tests (for pre-commit hooks)
test-short:
    go test -short ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run Go vet (accepts file list or defaults to all)
vet *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        go vet ./...
    else
        # Extract directories from file list for vet
        dirs=$(echo {{files}} | xargs -n1 dirname | sort -u | sed 's|^\./||')
        for dir in $dirs; do
            go vet ./$dir
        done
    fi

# === Formatting ===

# Format Go code (accepts file list or defaults to all)
fmt-go *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./...
    else
        go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt {{files}}
    fi

# Format YAML files (accepts file list or defaults to all)
fmt-yaml *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt .
    else
        go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt {{files}}
    fi

# Fix markdown files (accepts file list or defaults to all)
fmt-md *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        npm run fix:md
    else
        npx --no-install markdownlint-cli2 --fix {{files}}
    fi

# Fix trailing spaces and ensure newline at EOF (accepts file list or defaults to all)
fmt-whitespace *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        # Remove trailing spaces
        find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "*.txt" -o -name "*.json" -o -name "*.toml" -o -name "*.mod" -o -name "*.sum" -o -name "justfile" \) \
            -not -path "./vendor/*" -not -path "./.git/*" -not -path "./bin/*" \
            -exec perl -i -pe 's/[ \t]+$//' {} \;
        # Ensure newline at EOF
        find . -type f \( -name "*.go" -o -name "*.md" -o -name "*.yml" -o -name "*.yaml" -o -name "*.txt" -o -name "*.json" -o -name "*.toml" -o -name "*.mod" -o -name "*.sum" -o -name "justfile" \) \
            -not -path "./vendor/*" -not -path "./.git/*" -not -path "./bin/*" \
            -exec sh -c '[ "$(tail -c1 "$1")" != "" ] && echo >> "$1" || true' _ {} \;
    else
        # Process specific files
        for file in {{files}}; do
            perl -i -pe 's/[ \t]+$//' "$file"
            # Add newline at EOF only if missing
            [ "$(tail -c1 "$file")" != "" ] && echo >> "$file" || true
        done
    fi

# Format all code
fmt: fmt-whitespace fmt-go fmt-yaml fmt-md

# === Linting ===

# Lint Go code (accepts file list or defaults to all)
lint-go *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint run
    else
        go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint run {{files}}
    fi

# Lint markdown files (accepts file list or defaults to all)
lint-md *files:
    #!/usr/bin/env bash
    if [ -z "{{files}}" ]; then
        npm run lint:md
    else
        npx --no-install markdownlint-cli2 {{files}}
    fi

# Lint all code
lint: lint-go lint-md

# === Combined Commands ===

# Check code (format + lint + vet) - matches pre-commit hooks
check: fmt lint vet

# Check formatting without fixing (for CI)
check-fmt:
    #!/usr/bin/env bash
    # Check Go formatting
    if ! go run -mod=readonly github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt ./... --dry-run; then
        echo "Go formatting issues found. Run 'just fmt-go'"
        exit 1
    fi
    # Check YAML formatting
    if ! go run -mod=readonly github.com/google/yamlfmt/cmd/yamlfmt -dry .; then
        echo "YAML formatting issues found. Run 'just fmt-yaml'"
        exit 1
    fi
    # Check markdown formatting
    if ! npx --no-install markdownlint-cli2 '**/*.md' '#node_modules'; then
        echo "Markdown formatting issues found. Run 'just fmt-md'"
        exit 1
    fi
    echo "All formatting checks passed!"

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
