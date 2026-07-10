# Performance

Guide for measuring, monitoring, and optimizing TerraTidy performance.

!!! tip "User Guide"
    For end-user performance tips (parallel execution, `--changed` flag, caching), see [User Guide: Performance](../user-guide/performance.md).

## Benchmarks

TerraTidy uses Go's built-in benchmarking framework with per-package benchmark files (`*_benchmark_test.go`).

### Running Benchmarks

```bash
# Run all benchmarks
mise run benchmark

# Or manually with custom options
go test -bench=. -benchmem -benchtime=5s -run=^$ ./internal/...
```

The `mise run benchmark` task:

1. Runs all benchmarks with 5s duration and memory profiling
2. Saves results to `benchmarks/benchmark-YYYYMMDD-HHMMSS.txt`
3. If benchstat is installed, compares with the previous run

### Baseline

The official baseline is stored at `benchmarks/baseline.txt`. It was captured on:

- **Hardware:** Apple M2 Pro
- **OS:** darwin/arm64
- **Go version:** 1.26

The baseline serves as a reference point for CI regression detection.

### Benchmark Coverage

| Package | File | Benchmarks |
| ------- | ---- | ---------- |
| `internal/annotations` | `annotations_benchmark_test.go` | Parse, FilterFindings, IsSuppressed, RuleMatches |
| `internal/cache` | `cache_benchmark_test.go` | CacheHit, CacheMiss, CacheGetOrParse |
| `internal/config` | `config_benchmark_test.go` | LoadConfig, LoadConfigWithProfiles, LoadConfigWithImports, ApplyProfile |
| `internal/engines/format` | `format_benchmark_test.go` | FormatEngine, FormatEngineWithWrite, FormatFileCount, FormatLargeFile |
| `internal/engines/lint` | `lint_coverage_test.go` | LintModule, LintLargeModule |
| `internal/engines/policy` | `policy_benchmark_test.go` | PolicyEngine (simple, medium, complex), MultiFile, InvalidHCL |
| `internal/engines/style` | `style_benchmark_test.go` | StyleEngine (7 variants), DeepNesting |
| `internal/engines/style/rules` | `rules_benchmark_test.go` | Individual rules (10 benchmarks) |
| `internal/lsp` | `lsp_benchmark_test.go` | GetDiagnostics (4 variants) |
| `internal/output` | `output_benchmark_test.go` | SARIF, HTML, JSON, Text, JUnit, Markdown, Table, GitHubActions, ManyFindings |
| `internal/plugins` | `plugin_benchmark_test.go` | LoadYAMLRule, PluginManagerLoad |
| `internal/runner` | `runner_benchmark_test.go` | Sequential, Parallel, SingleEngine, MultipleEngines |
| `internal/vcs` | `git_benchmark_test.go` | IsGitRepo, GetChangedFiles, GetStagedFiles |
| `pkg/sdk` | `files_benchmark_test.go` | GroupFilesByDirectory, IsHCLFile, FileDiscoveryWorkflow |
| `pkg/sdk` | `types_benchmark_test.go` | ParseSeverity, SeverityLevel, SeverityCompare, LocationFromRange |

## Interpreting Results

### benchstat Output

The benchmark comparison uses [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Compare two runs
benchstat baseline.txt current.txt
```

Example output:

```text
                          │ baseline.txt │            current.txt             │
                          │    sec/op    │   sec/op     vs base               │
CacheHit-12                  15.0µs ± 9%    14.2µs ± 5%   -5.33% (p=0.002 n=6)
CacheMiss-12                  542µs ± 3%     525µs ± 2%   -3.14% (p=0.004 n=6)
FormatEngine-12               850µs ± 5%     920µs ± 8%   +8.24% (p=0.015 n=6)
```

Key columns:

- **sec/op**: Time per operation (lower is better)
- **vs base**: Percentage change (negative = faster, positive = slower)
- **p-value**: Statistical significance (p < 0.05 is significant)
- **±**: Standard deviation across runs

### Reading Memory Stats

```text
BenchmarkCacheHit-12    71050    15007 ns/op    3040 B/op    20 allocs/op
```

- `71050`: Number of iterations
- `15007 ns/op`: Time per operation (15µs)
- `3040 B/op`: Bytes allocated per operation
- `20 allocs/op`: Number of allocations per operation

## CI Regression Detection

The `.github/workflows/benchmark.yml` workflow:

1. **Triggers**: On main push (when baseline changes) or PRs with `benchmark` label
2. **Runs**: All benchmarks with `-count=6` for statistical validity
3. **Compares**: Current results against `benchmarks/baseline.txt`
4. **Threshold**: **15%** regression triggers a warning
5. **Comments**: Posts results to the PR with comparison

### Adding Benchmark Label

To run benchmarks on a PR, add the `benchmark` label. The workflow will:

- Run benchmarks
- Compare with baseline
- Post a comment with results
- Fail if regression exceeds 15%

### Updating Baseline

When you intentionally change performance characteristics:

```bash
# Run benchmarks and save new baseline
go test -bench=. -benchmem -count=6 -run=^$ ./internal/... > benchmarks/baseline.txt

# Commit the new baseline
git add benchmarks/baseline.txt
git commit -m "perf: update benchmark baseline"
```

## Profiling

Use mise tasks for common profiling workflows:

```bash
# CPU profile (default: BenchmarkRunnerParallel)
mise run profile:cpu

# Memory profile (default: BenchmarkRunnerParallel)
mise run profile:mem

# Profile a specific benchmark
mise run profile:cpu BenchmarkFormatEngine ./internal/engines/format/...
mise run profile:mem BenchmarkStyleEngine ./internal/engines/style/...
```

After profiling, view results in browser:

```bash
go tool pprof -http=:8080 cpu.prof
go tool pprof -http=:8080 mem.prof
```

### Manual Profiling

For more control, use go test directly:

```bash
# CPU profile
go test -bench=BenchmarkRunnerParallel -cpuprofile=cpu.prof -run=^$ ./internal/runner/...

# Memory profile
go test -bench=BenchmarkRunnerParallel -memprofile=mem.prof -run=^$ ./internal/runner/...

# Block profile (goroutine contention)
go test -bench=BenchmarkRunnerParallel -blockprofile=block.prof -run=^$ ./internal/runner/...
```

### Interpreting Profiles

The pprof web UI (`go tool pprof -http=:8080`) provides several views:

**Top View** - Functions sorted by resource usage:

- **flat**: Time/memory spent in the function itself (not callees)
- **cum** (cumulative): Time/memory spent in the function and all its callees
- **sum%**: Running total percentage

Look for functions with high `flat` values first, as these are direct optimization targets.

**Graph View** - Call graph with edge weights:

- Box size indicates resource usage
- Edge thickness shows call frequency
- Red nodes are hot paths

**Flame Graph** - Stack-based visualization:

- Width indicates time/memory
- Taller stacks show deep call chains
- Look for wide boxes (expensive operations)

**CPU Profile Indicators**:

| Pattern | Meaning | Action |
| ------- | ------- | ------ |
| High `flat` in `runtime.mallocgc` | Excessive allocations | Reduce allocs, use sync.Pool |
| High `flat` in `runtime.chanrecv` | Channel contention | Increase buffer or redesign |
| High `cum` but low `flat` | Slow callees | Drill into child functions |
| Wide flame in `encoding/json` | JSON overhead | Consider faster serializer |

**Memory Profile Modes**:

```bash
# View by allocation count (find frequent allocations)
go tool pprof -alloc_objects mem.prof

# View by allocation size (find large allocations)
go tool pprof -alloc_space mem.prof

# View current heap (find memory leaks)
go tool pprof -inuse_space mem.prof
```

**Memory Profile Indicators**:

| Pattern | Meaning | Action |
| ------- | ------- | ------ |
| Many small allocs in loop | Per-iteration allocation | Preallocate, reuse buffers |
| Large allocs in `[]byte` | String/slice growth | Use `bytes.Buffer`, preallocate |
| Growing `inuse_space` over time | Memory leak | Check for retained references |
| High allocs in `fmt.Sprintf` | String formatting | Use `strings.Builder` |

**Block Profile Indicators**:

| Pattern | Meaning | Action |
| ------- | ------- | ------ |
| High time in `sync.Mutex.Lock` | Lock contention | Use RWMutex, reduce critical section |
| High time in channel ops | Channel bottleneck | Buffer channels, use select |

## Performance Targets

### Hot Path Allocation Targets

These hot paths should minimize allocations:

| Operation | Target |
| --------- | ------ |
| `IsSuppressed` check | 0 allocs |
| `RuleMatches` | 0 allocs |
| `IsHCLFile` | 0 allocs |
| `SeverityLevel` | 0 allocs |
| `ParseSeverity` (valid) | 0-1 allocs |

### Throughput Targets

Based on current baseline (Apple M2 Pro):

| Operation | Target |
| --------- | ------ |
| Cache hit | < 20µs |
| Format single file | < 100µs |
| Style check (simple) | < 500µs |
| Config load | < 50µs |
| Profile apply | < 100ns |

### Scale Characteristics

- Format engine: Linear with file count (~80µs/file)
- Style engine: Linear with file count, superlinear with nesting depth
- Output formatters: HTML is slowest (~22ms/1000 findings), others ~0.3-1.6ms

## Writing Benchmarks

### Benchmark Template

```go
func BenchmarkOperation(b *testing.B) {
    // Setup (not timed)
    data := setupTestData()

    b.ResetTimer() // Start timing here
    for i := 0; i < b.N; i++ {
        _ = Operation(data)
    }
}
```

### Sub-benchmarks for Scale

```go
func BenchmarkOperation(b *testing.B) {
    sizes := []int{10, 100, 1000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := generateData(size)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = Operation(data)
            }
        })
    }
}
```

### Memory Benchmarks

For allocation-sensitive code, verify allocations in tests:

```go
func TestOperationAllocs(t *testing.T) {
    data := setupTestData()

    allocs := testing.AllocsPerRun(100, func() {
        _ = Operation(data)
    })

    if allocs > 0 {
        t.Errorf("expected 0 allocations, got %v", allocs)
    }
}
```

## Optimization Checklist

Before optimizing, verify:

- [ ] Is this actually a hot path? Profile first.
- [ ] Is the benchmark representative? Test with real data.
- [ ] Does optimization maintain correctness? Tests still pass.

Common optimizations:

1. **Reduce allocations**: Reuse buffers, avoid string concatenation
2. **Minimize locks**: Use `sync.RWMutex` for read-heavy workloads
3. **Batch operations**: Group I/O and network calls
4. **Cache computed values**: Store results of expensive operations
5. **Use sync.Pool**: For frequently allocated temporary objects
