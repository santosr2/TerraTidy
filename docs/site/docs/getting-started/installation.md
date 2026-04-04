# Installation

TerraTidy can be installed in several ways.

## Go Install

If you have Go installed (1.26.1+):

```bash
go install github.com/santosr2/TerraTidy/cmd/terratidy@latest
```

## Homebrew (macOS/Linux)

```bash
brew tap santosr2/tap https://github.com/santosr2/TerraTidy
brew install santosr2/tap/terratidy
```

## Download Binary

Download pre-built binaries from the [GitHub Releases](https://github.com/santosr2/TerraTidy/releases) page.

### Linux (amd64)

```bash
curl -LO https://github.com/santosr2/TerraTidy/releases/latest/download/terratidy_linux_amd64.tar.gz
tar xzf terratidy_linux_amd64.tar.gz
sudo mv terratidy /usr/local/bin/
```

### macOS (Apple Silicon)

```bash
curl -LO https://github.com/santosr2/TerraTidy/releases/latest/download/terratidy_darwin_arm64.tar.gz
tar xzf terratidy_darwin_arm64.tar.gz
sudo mv terratidy /usr/local/bin/
```

### Windows

```powershell
# Download from releases page
# Add to PATH
```

## Docker

```bash
# latest always points to the most recent stable release
docker pull ghcr.io/santosr2/terratidy:latest

# Pin to a specific version (recommended)
docker pull ghcr.io/santosr2/terratidy:v0.2.0-alpha.4

docker run --rm -v $(pwd):/app ghcr.io/santosr2/terratidy check
```

## Verify Installation

```bash
terratidy version
```

Every release includes checksums, cosign signatures, SBOMs, and build provenance attestations.
See [Verification](verification.md) for details on how to verify artifact integrity.

## Next Steps

- [Quick Start](quickstart.md) - Run your first checks
- [Configuration](configuration.md) - Configure TerraTidy for your project
