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

**Default behavior:** The config default is `parallel: true`, but the `--parallel` CLI flag
is additive (it enables parallel when your config has it disabled). If you haven't changed
the default config, engines already run in parallel.

**Note:** `fail_fast` only works in sequential mode. When using `--parallel`, all engines
run to completion regardless of errors.

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
go test -bench=. -benchmem ./internal/benchmark/
```

Available benchmarks:

- `BenchmarkFmtEngine` - Format engine throughput
- `BenchmarkStyleEngine` - Style checking throughput
- `BenchmarkRunnerSequential` vs `BenchmarkRunnerParallel` - Runner comparison
- `BenchmarkCacheHit` vs `BenchmarkCacheMiss` - Cache effectiveness
- `BenchmarkFileCount` - Scalability with 1, 5, 10, 25, 50 files

Compare results across runs:

```bash
# Save baseline
go test -bench=. -benchmem ./internal/benchmark/ > bench-before.txt

# Make changes, then compare
go test -bench=. -benchmem ./internal/benchmark/ > bench-after.txt
benchstat bench-before.txt bench-after.txt
```
