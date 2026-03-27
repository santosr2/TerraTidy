# Go Rule Example

A Go plugin rule that checks resources for a `tags` attribute.

## Build

```bash
cd examples/go-rule
go mod init require-tags
go mod tidy
go build -buildmode=plugin -o require-tags.so
```

## Install

```bash
mkdir -p ~/.terratidy/plugins
cp require-tags.so ~/.terratidy/plugins/
```

## Configuration

Enable plugins in `.terratidy.yaml`:

```yaml
plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
```
