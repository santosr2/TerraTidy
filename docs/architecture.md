# Architecture

## Overview

TerraTidy uses a library-first architecture where all tools are imported as Go libraries rather than shelling out to external CLIs.

## Design Principles

1. **Library-First**: Use Go libraries (hclwrite, tflint, OPA) instead of CLI tools
2. **Single Binary**: All dependencies vendored, no external requirements
3. **Extensible**: Plugin system for custom rules
4. **Performance**: Parallel processing, caching, minimal overhead
5. **Developer Experience**: Clear errors, interactive tools, hot-reload

## Components

### CLI Layer (`cmd/terratidy/`)

- Cobra-based CLI with subcommands
- Flag parsing and validation
- Output formatting

### Configuration (`internal/config/`)

- YAML parsing with imports
- Profile management
- Config merging and validation

### Engines (`internal/engines/`)

#### Fmt Engine

- Uses `github.com/hashicorp/hcl/v2/hclwrite`
- Formats .tf and .hcl files
- No terraform CLI needed

#### Style Engine

- HCL AST parsing
- Custom style rules
- Autofix capability

#### Lint Engine

- Imports `github.com/terraform-linters/tflint`
- Direct API calls, no subprocess
- Plugin support

#### Policy Engine

- Uses `github.com/open-policy-agent/opa/rego`
- Evaluates Rego policies
- Conftest-compatible

### Plugin System (`internal/plugins/`)

Supports three rule formats:

1. **Go Plugins**: Compiled `.so` files with full API access
2. **YAML Rules**: Declarative pattern matching
3. **Bash Scripts**: External tools with JSON output

### Output Formatters (`internal/output/`)

- Text (human-readable, colored)
- JSON (structured)
- SARIF (GitHub Code Scanning)

## Data Flow

```text
User Command
    ↓
CLI Parser
    ↓
Config Loader (with imports/profiles)
    ↓
Runner (orchestrates engines)
    ↓
Engines (parallel execution)
    ├─ Fmt Engine → hclwrite
    ├─ Style Engine → HCL AST + Rules
    ├─ Lint Engine → TFLint library
    └─ Policy Engine → OPA
    ↓
Findings Collection
    ↓
Output Formatter
    ↓
Exit Code
```

## Performance Optimizations

- **Parallel Processing**: Files processed concurrently
- **Worker Pools**: Bounded concurrency for I/O
- **Caching**: Parsed HCL files cached
- **Early Exit**: Stop on first error with `--fail-fast`

## Library Integration

### Why Libraries vs CLI?

| Aspect | Library | CLI Subprocess |
|--------|---------|----------------|
| Speed | ~1ms | ~50-100ms |
| Errors | Go stack traces | stderr parsing |
| Offline | ✅ Works | ❌ Needs binaries |
| Testing | Easy mocking | Complex fixtures |

### HCL Write Integration

```go
import "github.com/hashicorp/hcl/v2/hclwrite"

file, _ := hclwrite.ParseConfig(src, filename, hcl.Pos{})
formatted := file.Bytes() // Already formatted!
```

### TFLint Integration

```go
import "github.com/terraform-linters/tflint/tflint"

runner, _ := tflint.NewRunner(config)
issues, _ := runner.Run()
```

### OPA Integration

```go
import "github.com/open-policy-agent/opa/rego"

query, _ := rego.New(
    rego.Query("data.terraform.deny"),
    rego.Load([]string{policyDir}, nil),
).PrepareForEval(ctx)
```

## Extensibility

### SDK Package (`pkg/sdk/`)

Public API for rule authors:

```go
type Rule interface {
    Name() string
    Check(ctx *Context, file *hcl.File) ([]Finding, error)
    Fix(ctx *Context, file *hcl.File) ([]byte, error)
}
```

### Plugin Discovery

1. Check `.terratidy/rules/` directories
2. Load based on file extension (.so, .yaml, .sh)
3. Register with plugin manager
4. Execute during engine runs

## Testing Strategy

- **Unit Tests**: Each component isolated
- **Integration Tests**: End-to-end CLI commands
- **Fixture Tests**: Real Terraform code samples
- **Benchmarks**: Performance-critical paths

## Future Enhancements

- Language Server Protocol (LSP) support
- Real-time checking in editors
- Incremental analysis (only changed files)
- Distributed caching for monorepos
