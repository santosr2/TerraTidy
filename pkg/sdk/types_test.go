package sdk

import (
	"log"
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{"error severity", SeverityError, "error"},
		{"warning severity", SeverityWarning, "warning"},
		{"info severity", SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, Severity(tt.expected), tt.severity)
		})
	}
}

func TestFinding(t *testing.T) {
	t.Run("basic finding", func(t *testing.T) {
		finding := Finding{
			Rule:     "test-rule",
			Message:  "Test message",
			File:     "main.tf",
			Severity: SeverityError,
			Fixable:  true,
			Location: hcl.Range{
				Filename: "main.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 10, Byte: 9},
			},
		}

		assert.Equal(t, "test-rule", finding.Rule)
		assert.Equal(t, "Test message", finding.Message)
		assert.Equal(t, "main.tf", finding.File)
		assert.Equal(t, SeverityError, finding.Severity)
		assert.True(t, finding.Fixable)
		assert.Equal(t, 1, finding.Location.Start.Line)
	})

	t.Run("finding with fix function", func(t *testing.T) {
		fixCalled := false
		finding := Finding{
			Rule:    "fixable-rule",
			Fixable: true,
			FixFunc: func() ([]byte, error) {
				fixCalled = true
				return []byte("fixed content"), nil
			},
		}

		assert.NotNil(t, finding.FixFunc)
		result, err := finding.FixFunc()
		assert.NoError(t, err)
		assert.True(t, fixCalled)
		assert.Equal(t, []byte("fixed content"), result)
	})
}

func TestContext(t *testing.T) {
	t.Run("basic context", func(t *testing.T) {
		ctx := &Context{
			Config:  map[string]interface{}{"key": "value"},
			WorkDir: "/tmp/test",
			File:    "main.tf",
		}

		assert.Equal(t, "value", ctx.Config["key"])
		assert.Equal(t, "/tmp/test", ctx.WorkDir)
		assert.Equal(t, "main.tf", ctx.File)
	})

	t.Run("context with logger", func(t *testing.T) {
		logger := log.New(os.Stderr, "test: ", log.LstdFlags)
		ctx := &Context{
			Logger:  logger,
			WorkDir: ".",
		}

		assert.NotNil(t, ctx.Logger)
		assert.Equal(t, ".", ctx.WorkDir)
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := &Context{}

		assert.Nil(t, ctx.Config)
		assert.Nil(t, ctx.Logger)
		assert.Empty(t, ctx.WorkDir)
		assert.Empty(t, ctx.File)
	})
}

// MockRule implements the Rule interface for testing
type MockRule struct {
	name        string
	description string
	checkFunc   func(*Context, *hcl.File) ([]Finding, error)
	fixFunc     func(*Context, *hcl.File) ([]byte, error)
}

func (r *MockRule) Name() string        { return r.name }
func (r *MockRule) Description() string { return r.description }
func (r *MockRule) Check(ctx *Context, file *hcl.File) ([]Finding, error) {
	if r.checkFunc != nil {
		return r.checkFunc(ctx, file)
	}
	return nil, nil
}
func (r *MockRule) Fix(ctx *Context, file *hcl.File) ([]byte, error) {
	if r.fixFunc != nil {
		return r.fixFunc(ctx, file)
	}
	return nil, nil
}

func TestRuleInterface(t *testing.T) {
	t.Run("mock rule implementation", func(t *testing.T) {
		rule := &MockRule{
			name:        "mock-rule",
			description: "A mock rule for testing",
			checkFunc: func(_ *Context, _ *hcl.File) ([]Finding, error) {
				return []Finding{
					{Rule: "mock-rule", Message: "Found issue"},
				}, nil
			},
			fixFunc: func(_ *Context, _ *hcl.File) ([]byte, error) {
				return []byte("fixed"), nil
			},
		}

		// Test interface compliance
		var _ Rule = rule

		assert.Equal(t, "mock-rule", rule.Name())
		assert.Equal(t, "A mock rule for testing", rule.Description())

		findings, err := rule.Check(nil, nil)
		assert.NoError(t, err)
		assert.Len(t, findings, 1)
		assert.Equal(t, "mock-rule", findings[0].Rule)

		fixed, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, []byte("fixed"), fixed)
	})

	t.Run("rule with nil functions", func(t *testing.T) {
		rule := &MockRule{
			name:        "empty-rule",
			description: "Rule with no implementation",
		}

		findings, err := rule.Check(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, findings)

		fixed, err := rule.Fix(nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, fixed)
	})
}

func TestSeverityComparison(t *testing.T) {
	t.Run("severity string comparison", func(t *testing.T) {
		assert.Equal(t, "error", string(SeverityError))
		assert.Equal(t, "warning", string(SeverityWarning))
		assert.Equal(t, "info", string(SeverityInfo))
	})

	t.Run("severity type conversion", func(t *testing.T) {
		s := Severity("custom")
		assert.Equal(t, "custom", string(s))
	})
}
