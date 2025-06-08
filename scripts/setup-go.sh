#!/bin/bash
# Setup script for AgentCave Go implementation

set -e

echo "🕳️ Setting up AgentCave Go implementation..."

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
echo "🔨 Building AgentCave..."
mkdir -p bin
go build -o bin/agentcave cmd/agentcave/main.go

if [ -f "bin/agentcave" ]; then
    echo "✅ Build successful!"
    echo ""
    echo "🚀 AgentCave is ready to use!"
    echo ""
    echo "Next steps:"
    echo "1. Add bin/ to your PATH or run: make install"
    echo "2. Initialize in your project: agentcave init"
    echo "3. Start the MCP server: agentcave serve"
    echo ""
    echo "For more information, see README-go.md"
else
    echo "❌ Build failed. Please check the error messages above."
    exit 1
fi