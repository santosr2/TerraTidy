# Installation Guide

## Package Managers

### Homebrew (macOS/Linux)

```bash
brew install santosr2/tap/terratidy
```

### Scoop (Windows)

```bash
scoop bucket add santosr2 https://github.com/santosr2/scoop-bucket
scoop install terratidy
```

## Binary Download

Download from [GitHub Releases](https://github.com/santosr2/terratidy/releases):

### Linux

```bash
curl -L https://github.com/santosr2/terratidy/releases/latest/download/terratidy-linux-amd64 -o terratidy
chmod +x terratidy
sudo mv terratidy /usr/local/bin/
```

### macOS

```bash
curl -L https://github.com/santosr2/terratidy/releases/latest/download/terratidy-darwin-arm64 -o terratidy
chmod +x terratidy
sudo mv terratidy /usr/local/bin/
```

### Windows

Download `terratidy-windows-amd64.exe` from releases and add to PATH.

## Docker

```bash
docker pull ghcr.io/santosr2/terratidy:latest
```

Usage:

```bash
docker run --rm -v $(pwd):/app -w /app ghcr.io/santosr2/terratidy check
```

## From Source

Requires Go 1.24+:

```bash
go install github.com/santosr2/terratidy/cmd/terratidy@latest
```

## Verification

Verify installation:

```bash
terratidy version
```

## Next Steps

- [Quick Start Guide](../README.md#quick-start)
- [Configuration](./configuration.md)

