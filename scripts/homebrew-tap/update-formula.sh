#!/usr/bin/env bash
set -euo pipefail

# Script to update Homebrew formula with new release information
# Usage: ./update-formula.sh <version>

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.0"
    exit 1
fi

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

echo "Updating Homebrew formula for version $VERSION..."

# Download checksums
CHECKSUMS_URL="https://github.com/choplin/amux/releases/download/v${VERSION}/amux_${VERSION}_checksums.txt"
echo "Downloading checksums from $CHECKSUMS_URL..."

if ! curl -sL "$CHECKSUMS_URL" -o /tmp/amux_checksums.txt; then
    echo "Error: Failed to download checksums. Make sure version $VERSION is released."
    exit 1
fi

# Extract SHA256 values
DARWIN_AMD64_SHA=$(grep "darwin_amd64.tar.gz" /tmp/amux_checksums.txt | cut -d' ' -f1)
DARWIN_ARM64_SHA=$(grep "darwin_arm64.tar.gz" /tmp/amux_checksums.txt | cut -d' ' -f1)
LINUX_AMD64_SHA=$(grep "linux_amd64.tar.gz" /tmp/amux_checksums.txt | cut -d' ' -f1)
LINUX_ARM64_SHA=$(grep "linux_arm64.tar.gz" /tmp/amux_checksums.txt | cut -d' ' -f1)

# Check if we got all checksums
if [ -z "$DARWIN_AMD64_SHA" ] || [ -z "$DARWIN_ARM64_SHA" ] || [ -z "$LINUX_AMD64_SHA" ] || [ -z "$LINUX_ARM64_SHA" ]; then
    echo "Error: Could not extract all required checksums"
    echo "Darwin AMD64: $DARWIN_AMD64_SHA"
    echo "Darwin ARM64: $DARWIN_ARM64_SHA"
    echo "Linux AMD64: $LINUX_AMD64_SHA"
    echo "Linux ARM64: $LINUX_ARM64_SHA"
    exit 1
fi

# Create formula from template
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_FILE="$SCRIPT_DIR/amux.rb.template"
OUTPUT_FILE="$SCRIPT_DIR/amux.rb"

if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "Error: Template file not found: $TEMPLATE_FILE"
    exit 1
fi

# Replace placeholders
sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{DARWIN_AMD64_SHA256}}/${DARWIN_AMD64_SHA}/g" \
    -e "s/{{DARWIN_ARM64_SHA256}}/${DARWIN_ARM64_SHA}/g" \
    -e "s/{{LINUX_AMD64_SHA256}}/${LINUX_AMD64_SHA}/g" \
    -e "s/{{LINUX_ARM64_SHA256}}/${LINUX_ARM64_SHA}/g" \
    "$TEMPLATE_FILE" > "$OUTPUT_FILE"

echo "Formula updated successfully!"
echo ""
echo "Checksums:"
echo "  Darwin AMD64: $DARWIN_AMD64_SHA"
echo "  Darwin ARM64: $DARWIN_ARM64_SHA"
echo "  Linux AMD64:  $LINUX_AMD64_SHA"
echo "  Linux ARM64:  $LINUX_ARM64_SHA"
echo ""
echo "Generated formula saved to: $OUTPUT_FILE"
echo ""
echo "Next steps:"
echo "1. Copy $OUTPUT_FILE to your homebrew-amux repository"
echo "2. Commit and push the changes"
echo "3. Test installation: brew install --build-from-source ./Formula/amux.rb"

# Clean up
rm -f /tmp/amux_checksums.txt