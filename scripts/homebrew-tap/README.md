# Homebrew Tap Setup

This directory contains the tools and templates for managing the Homebrew tap for amux.

## Overview

Since Homebrew taps must be separate repositories, this directory provides:

1. **Formula Template**: `amux.rb.template` - Template for the Homebrew formula
2. **Update Script**: `update-formula.sh` - Script to generate formula with correct checksums
3. **GitHub Action**: `.github/workflows/update-homebrew-tap.yml` - Automated updates on release

## Setting Up the Tap Repository

### 1. Create the Repository

Create a new repository named `homebrew-amux` under the `choplin` organization:

```bash
# Create and clone the new repository
git clone https://github.com/choplin/homebrew-amux.git
cd homebrew-amux

# Create the Formula directory
mkdir -p Formula

# Create README
cat > README.md << 'EOF'
# Homebrew Amux

Homebrew tap for [amux](https://github.com/choplin/amux).

## Installation

\`\`\`bash
brew tap choplin/amux
brew install amux
\`\`\`

## Development

This tap is automatically updated when new releases are published to the main amux repository.
EOF

git add .
git commit -m "feat: initial Homebrew tap setup"
git push origin main

```

### 2. Set Up Repository Secrets

In the main `amux` repository:

1. Go to Settings → Secrets and variables → Actions
2. Add a new repository secret:
   - **Name**: `HOMEBREW_TAP_TOKEN`
   - **Value**: A GitHub Personal Access Token with `repo` scope

This token allows the main repository to trigger updates in the tap repository.

### 3. Add Workflow to Tap Repository

Create `.github/workflows/update-formula.yml` in the `homebrew-amux` repository:

```yaml
name: Update Formula

on:
  repository_dispatch:
    types: [update-formula]
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Download formula
        run: |
          curl -sL "${{ github.event.client_payload.formula_url }}" -o Formula/amux.rb

      - name: Commit changes
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"

          if git diff --quiet Formula/amux.rb; then
            echo "No changes to formula"
            exit 0
          fi

          VERSION="${{ github.event.client_payload.version }}"
          git add Formula/amux.rb
          git commit -m "chore: update amux to v${VERSION}"
          git push origin main
```

## Usage

### Automatic Updates

When a new release is published in the main amux repository:

1. The `update-homebrew-tap.yml` workflow runs automatically
2. It generates the formula with correct checksums
3. It triggers the tap repository to update

### Manual Updates

To manually update the formula:

```bash
# From the amux repository
cd scripts/homebrew-tap
./update-formula.sh 0.1.0

# Copy the generated formula to the tap repository
cp amux.rb /path/to/homebrew-amux/Formula/

# Commit and push in the tap repository
cd /path/to/homebrew-amux
git add Formula/amux.rb
git commit -m "chore: update amux to v0.1.0"
git push origin main
```

### Testing

To test the formula locally:

```bash
# In the homebrew-amux repository
brew install --build-from-source ./Formula/amux.rb

# Or audit the formula
brew audit --strict ./Formula/amux.rb
```

## Maintenance

- Keep the formula template in sync with release artifacts
- Monitor Homebrew formula best practices
- Update dependencies as needed

## Troubleshooting

### Common Issues

1. **Checksum Mismatch**: Ensure the release assets are fully uploaded before updating
2. **Formula Syntax**: Use `brew audit` to check for issues
3. **Installation Failures**: Test on both Intel and Apple Silicon Macs

### Debug Commands

```bash
# Check formula syntax
brew audit --strict Formula/amux.rb

# Test installation
brew install --verbose --debug Formula/amux.rb

# Check tap configuration
brew tap-info choplin/amux
```
