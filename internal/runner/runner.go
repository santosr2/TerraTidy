// Package runner provides parallel execution capabilities for TerraTidy engines.
// It enables running multiple engines concurrently and aggregating their results.
package runner

import (
	"context"
	"sync"

	"github.com/santosr2/terratidy/pkg/sdk"
)

// Engine defines the interface that all engines must implement
type Engine interface {
	Name() string
	Run(ctx context.Context, files []string) ([]sdk.Finding, error)
}

// EngineResult holds the result from a single engine execution
type EngineResult struct {
	Engine   string
	Findings []sdk.Finding
	Error    error
}

// Runner executes multiple engines, optionally in parallel
type Runner struct {
	engines  []Engine
	parallel bool
}

// New creates a new Runner
func New() *Runner {
	return &Runner{
		engines:  make([]Engine, 0),
		parallel: false,
	}
}

// AddEngine adds an engine to the runner
func (r *Runner) AddEngine(engine Engine) *Runner {
	r.engines = append(r.engines, engine)
	return r
}

// SetParallel enables or disables parallel execution
func (r *Runner) SetParallel(parallel bool) *Runner {
	r.parallel = parallel
	return r
}

// Run executes all engines and returns their combined findings
func (r *Runner) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	if r.parallel {
		return r.runParallel(ctx, files)
	}
	return r.runSequential(ctx, files)
}

// RunWithResults executes all engines and returns detailed results per engine
func (r *Runner) RunWithResults(ctx context.Context, files []string) []EngineResult {
	if r.parallel {
		return r.runParallelWithResults(ctx, files)
	}
	return r.runSequentialWithResults(ctx, files)
}

// runSequential executes engines one after another
func (r *Runner) runSequential(ctx context.Context, files []string) ([]sdk.Finding, error) {
	var allFindings []sdk.Finding

	for _, engine := range r.engines {
		select {
		case <-ctx.Done():
			return allFindings, ctx.Err()
		default:
		}

		findings, err := engine.Run(ctx, files)
		if err != nil {
			return allFindings, err
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

// runSequentialWithResults executes engines sequentially with detailed results
func (r *Runner) runSequentialWithResults(ctx context.Context, files []string) []EngineResult {
	results := make([]EngineResult, 0, len(r.engines))

	for _, engine := range r.engines {
		select {
		case <-ctx.Done():
			results = append(results, EngineResult{
				Engine: engine.Name(),
				Error:  ctx.Err(),
			})
			return results
		default:
		}

		findings, err := engine.Run(ctx, files)
		results = append(results, EngineResult{
			Engine:   engine.Name(),
			Findings: findings,
			Error:    err,
		})

		if err != nil {
			return results
		}
	}

	return results
}

// runParallel executes all engines concurrently
func (r *Runner) runParallel(ctx context.Context, files []string) ([]sdk.Finding, error) {
	results := r.runParallelWithResults(ctx, files)

	var allFindings []sdk.Finding
	for _, result := range results {
		if result.Error != nil {
			return allFindings, result.Error
		}
		allFindings = append(allFindings, result.Findings...)
	}

	return allFindings, nil
}

// runParallelWithResults executes all engines concurrently with detailed results
func (r *Runner) runParallelWithResults(ctx context.Context, files []string) []EngineResult {
	var wg sync.WaitGroup
	resultsChan := make(chan EngineResult, len(r.engines))

	for _, engine := range r.engines {
		wg.Add(1)
		go func(eng Engine) {
			defer wg.Done()

			findings, err := eng.Run(ctx, files)
			resultsChan <- EngineResult{
				Engine:   eng.Name(),
				Findings: findings,
				Error:    err,
			}
		}(engine)
	}

	// Wait for all engines to complete
	wg.Wait()
	close(resultsChan)

	// Collect results in order received
	results := make([]EngineResult, 0, len(r.engines))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// EngineCount returns the number of registered engines
func (r *Runner) EngineCount() int {
	return len(r.engines)
}

// IsParallel returns whether parallel execution is enabled
func (r *Runner) IsParallel() bool {
	return r.parallel
}
