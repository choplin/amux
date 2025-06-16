---
sidebar_position: 1
---

# Installation

Get Amux up and running in under a minute.

## Homebrew (Recommended)

The fastest way to install Amux on macOS and Linux:

```bash
brew tap choplin/amux
brew install amux
```

## Binary Releases

Download pre-built binaries for your platform:

1. Visit the [releases page](https://github.com/choplin/amux/releases)
2. Download the appropriate binary for your system
3. Make it executable: `chmod +x amux`
4. Move to your PATH: `sudo mv amux /usr/local/bin/`

## From Source

Build from source if you want the latest development version:

```bash
# Clone the repository
git clone https://github.com/choplin/amux.git
cd amux

# Build with just (recommended)
just build

# Or with go directly
go build -o bin/amux cmd/amux/main.go

# Add to PATH
sudo cp bin/amux /usr/local/bin/
```

### Prerequisites for Building

- Go 1.22 or later
- [Just](https://github.com/casey/just) (optional, but recommended)

## Verify Installation

```bash
amux version
```

You should see output like:

```text
Amux version 0.1.0
```

## Next Steps

- [Quick Start](quick-start.md) - Create your first workspace
- [Workspace Management](../guides/workspaces.md) - Learn workspace operations in detail
