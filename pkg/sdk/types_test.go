package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("basic context", func(t *testing.T) {
		ctx := &Context{
			Context: context.Background(),
			Options: map[string]any{"key": "value"},
			WorkDir: "/tmp/test",
			File:    "main.tf",
		}

		assert.Equal(t, "value", ctx.Options["key"])
		assert.Equal(t, "/tmp/test", ctx.WorkDir)
		assert.Equal(t, "main.tf", ctx.File)
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := &Context{}

		assert.Nil(t, ctx.Options)
		assert.Empty(t, ctx.WorkDir)
		assert.Empty(t, ctx.File)
	})

	t.Run("context with cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		ctx := &Context{
			Context: cancelCtx,
			File:    "test.tf",
		}

		// Initially not canceled
		select {
		case <-ctx.Done():
			t.Fatal("context should not be done")
		default:
			// expected
		}

		// Cancel and verify
		cancel()
		select {
		case <-ctx.Done():
			// expected
		default:
			t.Fatal("context should be done after cancel")
		}
		assert.Equal(t, context.Canceled, ctx.Err())
	})

	t.Run("context with deadline", func(t *testing.T) {
		deadline := time.Now().Add(100 * time.Millisecond)
		deadlineCtx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		ctx := &Context{
			Context: deadlineCtx,
			File:    "test.tf",
		}

		dl, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.Equal(t, deadline, dl)
	})
}

// fakeRule implements the Rule interface for testing.
// This is a test fake (not a mock framework) - a minimal implementation for testing
// that the plugin system correctly handles Rule implementations.
type fakeRule struct {
	name        string
	description string
	checkFunc   func(*Context, *hcl.File) ([]Finding, error)
	fixFunc     func(*Context, *hcl.File) ([]byte, error)
}

func (r *fakeRule) Name() string        { return r.name }
func (r *fakeRule) Description() string { return r.description }
func (r *fakeRule) Check(ctx *Context, file *hcl.File) ([]Finding, error) {
	if r.checkFunc != nil {
		return r.checkFunc(ctx, file)
	}
	return nil, nil
}

func (r *fakeRule) Fix(ctx *Context, file *hcl.File) ([]byte, error) {
	if r.fixFunc != nil {
		return r.fixFunc(ctx, file)
	}
	return nil, nil
}

func TestRuleInterface(t *testing.T) {
	t.Run("mock rule implements Rule", func(t *testing.T) {
		rule := &fakeRule{
			name:        "mock-rule",
			description: "A mock rule for testing",
			checkFunc: func(_ *Context, _ *hcl.File) ([]Finding, error) {
				return []Finding{{Rule: "mock-rule", Message: "Found issue"}}, nil
			},
		}

		var _ Rule = rule

		assert.Equal(t, "mock-rule", rule.Name())
		assert.Equal(t, "A mock rule for testing", rule.Description())

		findings, err := rule.Check(nil, nil)
		assert.NoError(t, err)
		assert.Len(t, findings, 1)
	})

	t.Run("mock rule implements Fixer", func(t *testing.T) {
		rule := &fakeRule{
			name: "fixable-rule",
			fixFunc: func(_ *Context, _ *hcl.File) ([]byte, error) {
				return []byte("fixed"), nil
			},
		}

		var _ Fixer = rule

		fixed, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte("fixed"), fixed)
	})

	t.Run("rule with nil functions", func(t *testing.T) {
		rule := &fakeRule{name: "empty-rule"}

		findings, err := rule.Check(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, findings)

		fixed, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, fixed)
	})
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultSev Severity
		expected   Severity
	}{
		{"error", "error", SeverityWarning, SeverityError},
		{"warning", "warning", SeverityError, SeverityWarning},
		{"info", "info", SeverityError, SeverityInfo},
		{"uppercase", "ERROR", SeverityWarning, SeverityError},
		{"mixed case", "Warning", SeverityError, SeverityWarning},
		{"unknown defaults to warning", "critical", SeverityWarning, SeverityWarning},
		{"unknown defaults to error", "critical", SeverityError, SeverityError},
		{"empty defaults", "", SeverityWarning, SeverityWarning},
		{"valid input with empty default", "error", "", SeverityError},
		{"unknown with empty default", "unknown", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseSeverity(tt.input, tt.defaultSev))
		})
	}
}

func TestSeverityLevel(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected int
	}{
		{"error has highest level", SeverityError, 2},
		{"warning has middle level", SeverityWarning, 1},
		{"info has lowest level", SeverityInfo, 0},
		{"unknown severity returns 0", Severity("unknown"), 0},
		{"empty severity returns 0", Severity(""), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.Level())
		})
	}

	t.Run("level ordering for filtering", func(t *testing.T) {
		// Verify error > warning > info for severity filtering
		assert.Greater(t, SeverityError.Level(), SeverityWarning.Level())
		assert.Greater(t, SeverityWarning.Level(), SeverityInfo.Level())
	})
}
