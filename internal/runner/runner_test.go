package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// mockEngine is a test engine implementation
type mockEngine struct {
	name     string
	findings []sdk.Finding
	err      error
	runFunc  func(ctx context.Context, files []string) ([]sdk.Finding, error)
}

func (e *mockEngine) Name() string {
	return e.name
}

func (e *mockEngine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	if e.runFunc != nil {
		return e.runFunc(ctx, files)
	}

	if e.err != nil {
		return nil, e.err
	}

	return e.findings, nil
}

func TestRunnerSequential(t *testing.T) {
	ctx := context.Background()

	engine1 := &mockEngine{
		name:     "engine1",
		findings: []sdk.Finding{{Rule: "rule1", Message: "msg1"}},
	}
	engine2 := &mockEngine{
		name:     "engine2",
		findings: []sdk.Finding{{Rule: "rule2", Message: "msg2"}, {Rule: "rule3", Message: "msg3"}},
	}

	runner := New().AddEngine(engine1).AddEngine(engine2)

	findings, err := runner.Run(ctx, []string{"test.tf"})
	require.NoError(t, err)
	assert.Len(t, findings, 3)
}

func TestRunnerParallel(t *testing.T) {
	ctx := context.Background()

	// Both engines block on a gate channel.
	// If execution were sequential, started.Wait() would deadlock because
	// the second engine can't start until the first finishes.
	var started sync.WaitGroup
	started.Add(2)
	gate := make(chan struct{})

	engine1 := &mockEngine{
		name: "engine1",
		runFunc: func(_ context.Context, _ []string) ([]sdk.Finding, error) {
			started.Done()
			<-gate
			return []sdk.Finding{{Rule: "rule1"}}, nil
		},
	}
	engine2 := &mockEngine{
		name: "engine2",
		runFunc: func(_ context.Context, _ []string) ([]sdk.Finding, error) {
			started.Done()
			<-gate
			return []sdk.Finding{{Rule: "rule2"}}, nil
		},
	}

	runner := New().AddEngine(engine1).AddEngine(engine2).SetParallel(true)

	done := make(chan struct{})
	var findings []sdk.Finding
	var runErr error

	go func() {
		findings, runErr = runner.Run(ctx, []string{"test.tf"})
		close(done)
	}()

	// Both engines must have started (proves parallelism).
	// If sequential, this would deadlock.
	started.Wait()
	close(gate)

	<-done
	require.NoError(t, runErr)
	assert.Len(t, findings, 2)
}

func TestRunnerWithError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("engine error")

	engine1 := &mockEngine{name: "engine1", findings: []sdk.Finding{{Rule: "rule1"}}}
	engine2 := &mockEngine{name: "engine2", err: expectedErr}

	runner := New().AddEngine(engine1).AddEngine(engine2)

	_, err := runner.Run(ctx, []string{"test.tf"})
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestRunnerWithResults(t *testing.T) {
	ctx := context.Background()

	engine1 := &mockEngine{name: "engine1", findings: []sdk.Finding{{Rule: "rule1"}}}
	engine2 := &mockEngine{name: "engine2", findings: []sdk.Finding{{Rule: "rule2"}}}

	runner := New().AddEngine(engine1).AddEngine(engine2)

	results := runner.RunWithResults(ctx, []string{"test.tf"})
	require.Len(t, results, 2)
	assert.Equal(t, "engine1", results[0].Engine)
	assert.Equal(t, "engine2", results[1].Engine)
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	engine1 := &mockEngine{
		name: "engine1",
		runFunc: func(ctx context.Context, _ []string) ([]sdk.Finding, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil, nil
			}
		},
	}
	engine2 := &mockEngine{name: "engine2"}

	runner := New().AddEngine(engine1).AddEngine(engine2)

	cancel()

	_, err := runner.Run(ctx, []string{"test.tf"})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunnerEngineCount(t *testing.T) {
	runner := New()
	assert.Equal(t, 0, runner.EngineCount())

	runner.AddEngine(&mockEngine{name: "e1"})
	runner.AddEngine(&mockEngine{name: "e2"})
	assert.Equal(t, 2, runner.EngineCount())
}

func TestRunnerIsParallel(t *testing.T) {
	runner := New()
	assert.False(t, runner.IsParallel())

	runner.SetParallel(true)
	assert.True(t, runner.IsParallel())
}
