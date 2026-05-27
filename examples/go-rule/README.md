# Go Rule Example

A Go plugin rule that checks resources for a `tags` attribute.

## Plugin Contract

Go plugins must export two symbols:

1. **`PluginMetadata`** - A `*plugins.PluginMetadata` variable with plugin info
2. **`New`** - A `func() plugins.RulePlugin` constructor function

The plugin must implement the `sdk.Rule` interface (`Name`, `Description`, `Check` methods).
Optionally implement `sdk.Fixer` for auto-fix support.

**Note:** Go plugins require binary compatibility. Build the plugin with the same Go version
and TerraTidy dependency versions as the installed binary. The `replace` directive in `go.mod`
is for local development; remove it when building against a released version.

## Build

`go.sum` is gitignored (regenerated locally), so the first build needs `go mod tidy`:

```bash
cd examples/go-rule
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
