# Architecture

Technical overview of TerraTidy's internal architecture.

## High-Level Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  (cmd/terratidy - Cobra commands)                           │
├─────────────────────────────────────────────────────────────┤
│                      Core Orchestrator                       │
│  (internal/core - Engine coordination, parallel execution)  │
├─────────────────────────────────────────────────────────────┤
│                        Engine Layer                          │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │   Fmt   │ │  Style  │ │  Lint   │ │ Policy  │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘           │
├─────────────────────────────────────────────────────────────┤
│                       Plugin System                          │
│  (internal/plugins - Custom rule loading)                   │
├─────────────────────────────────────────────────────────────┤
│                         SDK Layer                            │
│  (pkg/sdk - Public API for plugins)                         │
└─────────────────────────────────────────────────────────────┘
```

## Directory Structure

```text
terratidy/
├── cmd/
│   └── terratidy/           # CLI entry points
│       ├── main.go          # Main entry
│       ├── root.go          # Root command
│       ├── check.go         # Check command
│       ├── fmt.go           # Format command
│       ├── style.go         # Style command
│       ├── lint.go          # Lint command
│       ├── policy.go        # Policy command
│       └── lsp.go           # LSP server command
├── internal/
│   ├── core/                # Core orchestration
│   │   ├── runner.go        # Engine runner
│   │   ├── config.go        # Configuration loading
│   │   └── output.go        # Output formatting
│   ├── engines/             # Engine implementations
│   │   ├── fmt/             # Format engine
│   │   ├── style/           # Style engine
│   │   ├── lint/            # Lint engine
│   │   └── policy/          # Policy engine
│   ├── lsp/                 # Language server
│   └── plugins/             # Plugin system
├── pkg/
│   └── sdk/                 # Public SDK
│       ├── engine.go        # Engine interface
│       ├── finding.go       # Finding types
│       └── context.go       # Rule context
└── docs/                    # Documentation
```

## Core Components

### Engine Interface

All engines implement the `Engine` interface:

```go
type Engine interface {
    // Name returns the engine identifier
    Name() string

    // Run executes the engine on the given files
    Run(ctx context.Context, files []string) ([]Finding, error)
}
```

### Finding Type

Findings represent issues detected by engines:

```go
type Finding struct {
    Rule     string      // Rule identifier
    Message  string      // Human-readable message
    File     string      // Source file path
    Location hcl.Range   // Line/column information
    Severity Severity    // Error, Warning, Info
    Engine   string      // Originating engine
    Fixable  bool        // Can be auto-fixed
    Fix      *Fix        // Optional fix data
}
```

### Runner

The runner coordinates engine execution:

```go
type Runner struct {
    config  *Config
    engines []Engine
}

func (r *Runner) Run(ctx context.Context, files []string) (*Result, error) {
    var allFindings []Finding

    if r.config.Parallel {
        // Run engines concurrently
        findings := r.runParallel(ctx, files)
        allFindings = append(allFindings, findings...)
    } else {
        // Run engines sequentially
        for _, engine := range r.engines {
            findings, err := engine.Run(ctx, files)
            if err != nil {
                return nil, err
            }
            allFindings = append(allFindings, findings...)
        }
    }

    return &Result{Findings: allFindings}, nil
}
```

## Engine Implementations

### Format Engine

Uses the HCL formatter:

```go
func (e *FmtEngine) Run(ctx context.Context, files []string) ([]Finding, error) {
    for _, file := range files {
        content, _ := os.ReadFile(file)
        formatted := hclwrite.Format(content)

        if !bytes.Equal(content, formatted) {
            findings = append(findings, Finding{
                Rule:    "fmt",
                Message: "File is not formatted",
                File:    file,
                Fixable: true,
            })
        }
    }
    return findings, nil
}
```

### Style Engine

Implements custom style rules:

```go
func (e *StyleEngine) Run(ctx context.Context, files []string) ([]Finding, error) {
    for _, file := range files {
        ast, _ := hclparse.ParseHCLFile(file)

        for _, rule := range e.rules {
            if e.config.IsRuleEnabled(rule.Name()) {
                findings = append(findings, rule.Check(ast)...)
            }
        }
    }
    return findings, nil
}
```

### Lint Engine

Integrates with TFLint:

```go
func (e *LintEngine) Run(ctx context.Context, files []string) ([]Finding, error) {
    // Group files by module
    modules := groupByModule(files)

    for _, module := range modules {
        // Run TFLint on module
        result, _ := tflint.Run(module.Path, e.tflintConfig)

        for _, issue := range result.Issues {
            findings = append(findings, convertIssue(issue))
        }
    }
    return findings, nil
}
```

### Policy Engine

Uses OPA for policy evaluation:

```go
func (e *PolicyEngine) Run(ctx context.Context, files []string) ([]Finding, error) {
    // Parse Terraform to JSON
    input := parseToJSON(files)

    // Load policies
    policies := loadPolicies(e.policyDirs)

    // Evaluate
    r := rego.New(
        rego.Query("data.terraform.deny"),
        rego.Module("policy.rego", policies),
        rego.Input(input),
    )

    rs, _ := r.Eval(ctx)
    return processResults(rs), nil
}
```

## Configuration System

### Configuration Loading

```go
func LoadConfig(path string) (*Config, error) {
    // Try explicit path
    if path != "" {
        return loadFromFile(path)
    }

    // Search for config file
    searchPaths := []string{
        ".terratidy.yaml",
        ".terratidy.yml",
        "terratidy.yaml",
    }

    for _, p := range searchPaths {
        if fileExists(p) {
            return loadFromFile(p)
        }
    }

    return DefaultConfig(), nil
}
```

### Profile Resolution

```go
func (c *Config) ResolveProfile(name string) *Config {
    profile, ok := c.Profiles[name]
    if !ok {
        return c
    }

    // Merge profile with base config
    merged := c.Clone()
    merged.Merge(profile)
    return merged
}
```

## Output System

### Formatter Interface

```go
type Formatter interface {
    Format(result *Result) ([]byte, error)
}
```

### Implementations

- `TextFormatter` - Human-readable colored output
- `JSONFormatter` - Machine-readable JSON
- `SARIFFormatter` - GitHub-compatible SARIF
- `HTMLFormatter` - Interactive HTML reports
- `JUnitFormatter` - CI-compatible JUnit XML

## LSP Server

### Architecture

```text
┌─────────────────────────────────────────┐
│              LSP Server                  │
├─────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────────┐  │
│  │   Handler   │  │  Document Store  │  │
│  └─────────────┘  └─────────────────┘  │
├─────────────────────────────────────────┤
│  ┌─────────────────────────────────────┐│
│  │        Engine Integration            ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

### Request Handling

```go
func (s *Server) handleTextDocumentDidChange(params DidChangeParams) {
    // Update document store
    s.documents.Update(params.URI, params.Changes)

    // Run diagnostics
    go s.publishDiagnostics(params.URI)
}

func (s *Server) publishDiagnostics(uri string) {
    content := s.documents.Get(uri)
    findings := s.runner.Run(context.Background(), []string{uri})

    diagnostics := convertToDiagnostics(findings)
    s.client.PublishDiagnostics(uri, diagnostics)
}
```

## Plugin System

### Plugin Loading

```go
func LoadPlugins(dirs []string) ([]Engine, error) {
    var plugins []Engine

    for _, dir := range dirs {
        files, _ := filepath.Glob(filepath.Join(dir, "*.so"))

        for _, file := range files {
            p, _ := plugin.Open(file)
            sym, _ := p.Lookup("Engine")
            engine := sym.(Engine)
            plugins = append(plugins, engine)
        }
    }

    return plugins, nil
}
```

## Performance Considerations

### Parallel Execution

- Engines run concurrently when `parallel: true`
- Files are grouped by module for efficiency
- Context cancellation for early termination

### Caching

- Parsed ASTs are cached per file
- Policy compilation results are cached
- File checksums for incremental checking
