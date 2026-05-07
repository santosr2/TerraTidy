# Performance

Tips for optimizing TerraTidy performance on large repositories.

## Parallel Execution

Run engines concurrently with the `--parallel` flag:

```bash
terratidy check --parallel
```

Or enable in config:

```yaml
parallel: true
```

Parallel mode runs all enabled engines (fmt, style, lint, policy) simultaneously using
goroutines. This is most effective when multiple engines are enabled.

**Default behavior:** Parallel execution is on by default. When no config file is present,
or when `parallel` is omitted from config, TerraTidy runs engines in parallel. Use `--parallel`
to force parallel when config has explicitly disabled it, and `--no-parallel` to force
sequential execution regardless of config.

**Note:** `fail_fast` only works in sequential mode. When running in parallel, all engines
run to completion regardless of errors. Pair `--no-parallel` with `fail_fast: true` to stop
on the first engine that reports errors.

## Engine Selection

Only run the engines you need:

```bash
# Skip slow engines for local development
terratidy check --skip-lint --skip-policy

# Or use a lightweight profile
terratidy check --profile development
```

Typical engine speed (fastest to slowest): fmt > style > lint > policy.

## --changed Flag

Only check files modified since the default branch:

```bash
terratidy check --changed
```

This uses git to detect staged, unstaged, and untracked `.tf`/`.hcl`/`.tfvars` files.
On a 1000-file repo where you've changed 5 files, this can be 200x faster.

## Caching

TerraTidy automatically caches parsed HCL files to avoid redundant reads. The cache uses:

- **5-minute TTL** for entry expiration
- **1000-entry limit** with LRU eviction
- **Mod-time validation** to detect file changes

The cache is shared across engine runs within a single invocation. No configuration needed.

## Large Repositories

For repositories with thousands of Terraform files:

1. **Use `--changed` in CI** for PR checks (only check modified files)
2. **Enable parallel mode** for full repository scans
3. **Use profiles** to run fewer engines during development
4. **Split by directory** to check specific modules:

```bash
terratidy check ./modules/networking
terratidy check ./modules/compute
```

## Benchmarking

Run the built-in benchmarks to measure performance on your hardware:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/...

# Or use mise task
mise run benchmark
```

Benchmarks are located in their respective packages:

| Package | Benchmarks |
| ------- | ---------- |
| `internal/cache` | Cache hit/miss, GetOrParse |
| `internal/config` | Config loading, profiles, imports |
| `internal/engines/format` | Format engine, file count scaling |
| `internal/engines/lint` | Lint module, large module |
| `internal/engines/policy` | Policy evaluation, multi-file |
| `internal/engines/style` | Style engine configurations |
| `internal/engines/style/rules` | Individual rule performance |
| `internal/lsp` | Diagnostic computation |
| `internal/output` | SARIF, HTML, JSON, text output |
| `internal/plugins` | YAML rule loading, plugin manager |
| `internal/runner` | Sequential vs parallel, single vs multi-engine |
| `internal/vcs` | Git operations |

Compare results across runs:

```bash
# Save baseline
go test -bench=. -benchmem ./internal/... > bench-before.txt

# Make changes, then compare
go test -bench=. -benchmem ./internal/... > bench-after.txt
benchstat bench-before.txt bench-after.txt
```
