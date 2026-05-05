# Architecture

Technical overview of TerraTidy's internal architecture.

## High-Level Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                           │
│  (cmd/terratidy - Cobra commands)                           │
├─────────────────────────────────────────────────────────────┤
│                      Core Orchestrator                      │
│  (internal/runner - Engine coordination, parallel execution)│
├─────────────────────────────────────────────────────────────┤
│                        Engine Layer                         │
│       ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│       │   Fmt   │ │  Style  │ │  Lint   │ │ Policy  │       │
│       └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
├─────────────────────────────────────────────────────────────┤
│                       Plugin System                         │
│  (internal/plugins - Custom rule loading)                   │
├─────────────────────────────────────────────────────────────┤
│                         SDK Layer                           │
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
│   ├── runner/              # Engine runner, parallel execution
│   │   └── runner.go        # Engine interface, Runner struct
│   ├── config/              # Configuration loading
│   ├── output/              # Output formatting
│   ├── engines/             # Engine implementations
│   │   ├── fmt/             # Format engine
│   │   ├── style/           # Style engine
│   │   ├── lint/            # Lint engine
│   │   └── policy/          # Policy engine
│   ├── lsp/                 # Language server
│   └── plugins/             # Plugin system
├── pkg/
│   └── sdk/                 # Public SDK
│       └── types.go         # Rule interface, Finding, Context types
└── docs/                    # Documentation
```

## Core Components

### Engine Interface

All engines implement the `Engine` interface defined in `internal/runner/runner.go`:

```go
type Engine interface {
    Name() string
    Run(ctx context.Context, files []string) ([]sdk.Finding, error)
}
```

### Finding Type

Findings represent issues detected by engines:

```go
type Finding struct {
    Rule     string   `json:"rule"`
    Message  string   `json:"message"`
    File     string   `json:"file"`
    Location Location `json:"location"`
    Severity Severity `json:"severity"`
    Fixable  bool     `json:"fixable,omitempty"`
    IsDiff   bool     `json:"is_diff,omitempty"`
}
```

`Fixable` is set by the engine based on whether the rule implements `sdk.Fixer`.
The engine calls `Fixer.Fix(ctx, file)` lazily — only when applying fixes —
instead of asking rules to precompute fix bytes during `Check()`.

`IsDiff` is set by the fmt and style engines when `Message` contains a unified
diff; the CLI uses it to route diff content through a renderer instead of
printing it as plain text.

### Runner

The runner coordinates engine execution:

```go
type Runner struct {
    engines  []Engine
    parallel bool
}

func (r *Runner) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
    if r.parallel {
        return r.runParallel(ctx, files)
    }
    return r.runSequential(ctx, files)
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

Provides built-in AST rules and optional TFLint integration (subprocess, not linked):

```go
func (e *LintEngine) Run(ctx context.Context, files []string) ([]Finding, error) {
    // Run built-in AST rules first
    for _, file := range files {
        findings = append(findings, e.runBuiltinRules(file)...)
    }

    // Optionally invoke TFLint as subprocess (not embedded)
    if e.config.UseTFLint {
        modules := groupByModule(files)
        for _, module := range modules {
            cmd := exec.CommandContext(ctx, "tflint", "--format=json", module.Path)
            output, _ := cmd.Output()
            findings = append(findings, parseTFLintOutput(output)...)
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
func Load(path string) (*Config, error) {
    // Default to .terratidy.yaml
    if path == "" {
        path = ".terratidy.yaml"
    }

    // If file doesn't exist, return defaults
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return DefaultConfig(), nil
    }

    // Read, expand env vars, unmarshal YAML
    data, err := os.ReadFile(path)
    // ... expand ${VAR} and ${VAR:-default} syntax ...

    // Load imports (glob patterns)
    if len(cfg.Imports) > 0 {
        cfg.loadImports(filepath.Dir(path))
    }

    // Validate and return
    cfg.Validate()
    return &cfg, nil
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
    Format(findings []sdk.Finding, w io.Writer) error
}
```

### Implementations

- `TextFormatter` - Human-readable colored output
- `JSONFormatter` - Machine-readable JSON
- `SARIFFormatter` - GitHub-compatible SARIF
- `JUnitFormatter` - JUnit XML for CI systems
- `MarkdownFormatter` - Markdown tables
- `HTMLFormatter` - Interactive HTML reports
- `TableFormatter` - Tabular text output
- `GitHubActionsFormatter` - GitHub Actions annotations

## LSP Server

### Architecture

```text
┌─────────────────────────────────────────┐
│              LSP Server                 │
├─────────────────────────────────────────┤
│  ┌─────────────┐   ┌─────────────────┐  │
│  │   Handler   │   │  Document Store │  │
│  └─────────────┘   └─────────────────┘  │
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │        Engine Integration           │ │
│ └─────────────────────────────────────┘ │
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

### Debouncing

The server debounces diagnostics to prevent running expensive checks on every keystroke, with an independent timer per open document:

```go
const DefaultDebounceDelay = 500 * time.Millisecond

func (s *Server) scheduleDebouncedDiagnostics(uri string) {
    s.debounceMu.Lock()
    defer s.debounceMu.Unlock()

    // Cancel any pending timer for this URI
    if timer, ok := s.debounceTimers[uri]; ok {
        timer.Stop()
    }

    s.debounceTimers[uri] = time.AfterFunc(s.debounceDelay, func() {
        // Clean up timer from map before running diagnostics
        s.debounceMu.Lock()
        delete(s.debounceTimers, uri)
        s.debounceMu.Unlock()

        _ = s.publishDiagnostics(uri)
    })
}
```

### Config Auto-Reload

The server watches config files for changes and auto-reloads:

```text
fsnotify event
      │
      ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  scheduleConfig │────▶│   reloadConfig  │────▶│   republishAll  │
│  Reload (100ms) │     │   (engineMu)    │     │   Diagnostics   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
     debounce            load + swap              async via WaitGroup
```

Key features:

- Watches `.terratidy.yaml` and all imported config files
- Debounces rapid file events (100ms) to coalesce editor writes
- Swaps config and engines under `engineMu` to avoid partial state
- Republishes diagnostics asynchronously via `republishWg` WaitGroup
- On reload failure, keeps the previous config (bad save won't break session)
- Skips scheduling if server is shutting down (`closing` flag)

```go
const ConfigReloadDebounceDelay = 100 * time.Millisecond

// initConfigWatcher starts watching config files for changes (simplified)
func (s *Server) initConfigWatcher(configPath string, configFiles []string) error {
    watcher, _ := fsnotify.NewWatcher()
    for _, file := range configFiles {
        watcher.Add(file)  // errors logged, not fatal
    }
    go s.handleConfigWatchEvents()
    return nil
}

// scheduleConfigReload debounces rapid file events (simplified)
func (s *Server) scheduleConfigReload() {
    s.configWatcherMu.Lock()
    defer s.configWatcherMu.Unlock()

    if s.closing {
        return  // skip if shutting down
    }
    if s.configReloadTimer != nil {
        s.configReloadTimer.Stop()
    }
    s.configReloadTimer = time.AfterFunc(s.configReloadDelay, func() {
        if err := s.reloadConfig(); err == nil {
            s.republishWg.Go(s.republishAllDiagnostics)
        }
    })
}
```

The `workspace/didChangeConfiguration` notification triggers an immediate (non-debounced) config reload when the client pushes new settings.
Unlike the fsnotify path, diagnostics are not automatically republished.

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
