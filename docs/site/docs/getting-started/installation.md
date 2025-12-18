# Installation

TerraTidy can be installed in several ways.

## Go Install

If you have Go installed (1.25+):

```bash
go install github.com/santosr2/terratidy/cmd/terratidy@latest
```

## Homebrew (macOS/Linux)

```bash
brew install santosr2/tap/terratidy
```

## Download Binary

Download pre-built binaries from the [GitHub Releases](https://github.com/santosr2/terratidy/releases) page.

### Linux (amd64)

```bash
curl -LO https://github.com/santosr2/terratidy/releases/latest/download/terratidy_linux_amd64.tar.gz
tar xzf terratidy_linux_amd64.tar.gz
sudo mv terratidy /usr/local/bin/
```

### macOS (Apple Silicon)

```bash
curl -LO https://github.com/santosr2/terratidy/releases/latest/download/terratidy_darwin_arm64.tar.gz
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
docker pull ghcr.io/santosr2/terratidy:latest
docker run --rm -v $(pwd):/workspace ghcr.io/santosr2/terratidy check
```

## Verify Installation

```bash
terratidy version
```

## Next Steps

- [Quick Start](quickstart.md) - Run your first checks
- [Configuration](configuration.md) - Configure TerraTidy for your project
