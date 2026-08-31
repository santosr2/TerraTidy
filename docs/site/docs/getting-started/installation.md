# Installation

TerraTidy can be installed in several ways.

## Go Install

If you have Go installed (1.25+):

```bash
go install github.com/santosr2/TerraTidy/cmd/terratidy@v0.3.0
```

!!! tip "Pin your version"
    Pin to an explicit version tag for reproducible installs rather than tracking a
    floating reference.

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
# Download the latest release
Invoke-WebRequest -Uri "https://github.com/santosr2/TerraTidy/releases/latest/download/terratidy_windows_amd64.zip" -OutFile "terratidy.zip"

# Extract the archive
Expand-Archive -Path "terratidy.zip" -DestinationPath "."

# Move to a directory in your PATH (e.g., C:\Program Files\TerraTidy)
New-Item -ItemType Directory -Force -Path "C:\Program Files\TerraTidy"
Move-Item -Path "terratidy.exe" -Destination "C:\Program Files\TerraTidy\"

# Add to PATH (run as Administrator)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\TerraTidy", "Machine")
```

## Docker

```bash
# Pin to a specific version (recommended)
docker pull ghcr.io/santosr2/terratidy:v0.3.0

# Or track the most recent stable release
docker pull ghcr.io/santosr2/terratidy:latest

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
