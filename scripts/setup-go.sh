#!/bin/bash
# Setup script for Amux Go implementation

set -e

echo "🕳️ Setting up Amux Go implementation..."

# Check Go version
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ Found Go version: $GO_VERSION"

# Check git
if ! command -v git &> /dev/null; then
    echo "❌ Git is not installed. Please install Git."
    exit 1
fi
echo "✅ Git is installed"

# Initialize go modules
echo "📦 Initializing Go modules..."
go mod download
go mod tidy

# Build the binary
echo "🔨 Building Amux..."
mkdir -p bin
go build -o bin/amux cmd/amux/main.go

if [ -f "bin/amux" ]; then
    echo "✅ Build successful!"
    echo ""
    echo "🚀 Amux is ready to use!"
    echo ""
    echo "Next steps:"
    echo "1. Add bin/ to your PATH or run: make install"
    echo "2. Initialize in your project: amux init"
    echo "3. Start the MCP server: amux serve"
    echo ""
    echo "For more information, see README-go.md"
else
    echo "❌ Build failed. Please check the error messages above."
    exit 1
fi