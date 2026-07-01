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
cd ~/.terratidy/plugins && shasum -a 256 require-tags.so > .terratidy-plugins.sha256
```

The manifest must record the bare filename (not a path), which is why the `shasum` step `cd`s into the install directory first. See "Plugin Integrity" below.

## Configuration

Enable plugins in `.terratidy.yaml`:

```yaml
plugins:
  enabled: true
  directories:
    - ~/.terratidy/plugins
```

## Plugin Integrity

Unlike the bash example, the `.terratidy-plugins.sha256` manifest is **not** committed for Go plugins:
every `go build -buildmode=plugin` produces a different binary hash (Go bakes build metadata into the artifact),
so a committed manifest goes stale on the first local build and the loader rejects the plugin with a
checksum-mismatch error (Go plugin verification is a hard fail, not warn-only).

You must generate the manifest locally after every build, in the directory that holds the `.so`
(the loader keys on the basename, so the manifest's filename column has to be `require-tags.so`, not a path).
For the example test configs (which point at `examples/go-rule/` directly):

```bash
cd examples/go-rule
go build -buildmode=plugin -o require-tags.so
shasum -a 256 require-tags.so > .terratidy-plugins.sha256
```

CI runs the same steps in [.github/workflows/examples-test.yml](../../.github/workflows/examples-test.yml).
