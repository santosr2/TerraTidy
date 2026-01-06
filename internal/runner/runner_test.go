package runner

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/santosr2/terratidy/pkg/sdk"
)

// mockEngine is a test engine implementation
type mockEngine struct {
	name      string
	findings  []sdk.Finding
	err       error
	delay     time.Duration
	callCount *int32
}

func (e *mockEngine) Name() string {
	return e.name
}

func (e *mockEngine) Run(ctx context.Context, files []string) ([]sdk.Finding, error) {
	if e.callCount != nil {
		atomic.AddInt32(e.callCount, 1)
	}

	if e.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(e.delay):
		}
	}

	if e.err != nil {
		return nil, e.err
	}

	return e.findings, nil
}

func TestRunnerSequential(t *testing.T) {
	ctx := context.Background()

	engine1 := &mockEngine{
		name: "engine1",
		findings: []sdk.Finding{
			{Rule: "rule1", Message: "msg1"},
		},
	}
	engine2 := &mockEngine{
		name: "engine2",
		findings: []sdk.Finding{
			{Rule: "rule2", Message: "msg2"},
			{Rule: "rule3", Message: "msg3"},
		},
	}

	runner := New().
		AddEngine(engine1).
		AddEngine(engine2)

	findings, err := runner.Run(ctx, []string{"test.tf"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 3 {
		t.Errorf("expected 3 findings, got %d", len(findings))
	}
}

func TestRunnerParallel(t *testing.T) {
	ctx := context.Background()

	var count1, count2 int32
	engine1 := &mockEngine{
		name:      "engine1",
		findings:  []sdk.Finding{{Rule: "rule1"}},
		delay:     10 * time.Millisecond,
		callCount: &count1,
	}
	engine2 := &mockEngine{
		name:      "engine2",
		findings:  []sdk.Finding{{Rule: "rule2"}},
		delay:     10 * time.Millisecond,
		callCount: &count2,
	}

	runner := New().
		AddEngine(engine1).
		AddEngine(engine2).
		SetParallel(true)

	start := time.Now()
	findings, err := runner.Run(ctx, []string{"test.tf"})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}

	// Parallel execution should be faster than sequential
	// (20ms sequential vs ~10ms parallel)
	if duration > 20*time.Millisecond {
		t.Errorf("parallel execution took too long: %v", duration)
	}

	if atomic.LoadInt32(&count1) != 1 {
		t.Error("engine1 should have been called once")
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Error("engine2 should have been called once")
	}
}

func TestRunnerWithError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("engine error")

	engine1 := &mockEngine{
		name:     "engine1",
		findings: []sdk.Finding{{Rule: "rule1"}},
	}
	engine2 := &mockEngine{
		name: "engine2",
		err:  expectedErr,
	}

	runner := New().
		AddEngine(engine1).
		AddEngine(engine2)

	_, err := runner.Run(ctx, []string{"test.tf"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRunnerWithResults(t *testing.T) {
	ctx := context.Background()

	engine1 := &mockEngine{
		name:     "engine1",
		findings: []sdk.Finding{{Rule: "rule1"}},
	}
	engine2 := &mockEngine{
		name:     "engine2",
		findings: []sdk.Finding{{Rule: "rule2"}},
	}

	runner := New().
		AddEngine(engine1).
		AddEngine(engine2)

	results := runner.RunWithResults(ctx, []string{"test.tf"})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check results are in order for sequential execution
	if results[0].Engine != "engine1" {
		t.Errorf("expected first result from engine1, got %s", results[0].Engine)
	}
	if results[1].Engine != "engine2" {
		t.Errorf("expected second result from engine2, got %s", results[1].Engine)
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	engine1 := &mockEngine{
		name:  "engine1",
		delay: 100 * time.Millisecond,
	}
	engine2 := &mockEngine{
		name: "engine2",
	}

	runner := New().
		AddEngine(engine1).
		AddEngine(engine2)

	// Cancel immediately
	cancel()

	_, err := runner.Run(ctx, []string{"test.tf"})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRunnerEngineCount(t *testing.T) {
	runner := New()

	if runner.EngineCount() != 0 {
		t.Errorf("expected 0 engines, got %d", runner.EngineCount())
	}

	runner.AddEngine(&mockEngine{name: "e1"})
	runner.AddEngine(&mockEngine{name: "e2"})

	if runner.EngineCount() != 2 {
		t.Errorf("expected 2 engines, got %d", runner.EngineCount())
	}
}

func TestRunnerIsParallel(t *testing.T) {
	runner := New()

	if runner.IsParallel() {
		t.Error("expected parallel to be false by default")
	}

	runner.SetParallel(true)
	if !runner.IsParallel() {
		t.Error("expected parallel to be true after SetParallel(true)")
	}
}
