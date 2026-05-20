package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCmd(t *testing.T) {
	t.Run("command metadata", func(t *testing.T) {
		assert.Equal(t, "check [paths...]", checkCmd.Use)
		assert.Equal(t, "Run all checks (fmt, style, lint, policy)", checkCmd.Short)
		assert.NotEmpty(t, checkCmd.Long)
		assert.NotEmpty(t, checkCmd.Example)
	})

	t.Run("has skip-fmt flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("skip-fmt")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has skip-style flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("skip-style")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has skip-lint flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("skip-lint")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has skip-policy flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("skip-policy")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has parallel flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("parallel")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has no-parallel flag", func(t *testing.T) {
		flag := checkCmd.Flags().Lookup("no-parallel")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

// TestCheckCmd_MissingTargetFile verifies that passing a non-existent target path
// returns an ExitError with code 3 and a "file not found" message. Previously the
// CLI exited silently with code 3 and no message.
func TestCheckCmd_MissingTargetFile(t *testing.T) {
	resetCheckGlobals(t)

	// Construct a path inside a real temp dir that is guaranteed not to exist —
	// avoids races with anything that might write to /tmp.
	missing := filepath.Join(t.TempDir(), "does-not-exist.tf")

	format = "text"
	cfgFile = ""
	changed = false

	rootCmd.SetArgs([]string{"check", missing})
	err := rootCmd.Execute()

	require.Error(t, err, "check command should fail for missing target file")

	var exitErr *sdk.ExitError
	require.True(t, errors.As(err, &exitErr), "expected ExitError, got: %v", err)
	assert.Equal(t, sdk.ExitInternal, exitErr.Code, "exit code should be 3 for missing target file")
	assert.Contains(t, err.Error(), "file not found", "error message should say 'file not found'")
	assert.Contains(t, err.Error(), missing, "error message should include the missing path")
}

func TestCountBySeverity(t *testing.T) {
	tests := []struct {
		name         string
		findings     []sdk.Finding
		wantErrors   int
		wantWarnings int
		wantInfo     int
	}{
		{
			name:         "empty findings",
			findings:     []sdk.Finding{},
			wantErrors:   0,
			wantWarnings: 0,
			wantInfo:     0,
		},
		{
			name: "all errors",
			findings: []sdk.Finding{
				{Severity: sdk.SeverityError},
				{Severity: sdk.SeverityError},
			},
			wantErrors:   2,
			wantWarnings: 0,
			wantInfo:     0,
		},
		{
			name: "all warnings",
			findings: []sdk.Finding{
				{Severity: sdk.SeverityWarning},
				{Severity: sdk.SeverityWarning},
				{Severity: sdk.SeverityWarning},
			},
			wantErrors:   0,
			wantWarnings: 3,
			wantInfo:     0,
		},
		{
			name: "all info",
			findings: []sdk.Finding{
				{Severity: sdk.SeverityInfo},
			},
			wantErrors:   0,
			wantWarnings: 0,
			wantInfo:     1,
		},
		{
			name: "mixed severities",
			findings: []sdk.Finding{
				{Severity: sdk.SeverityError},
				{Severity: sdk.SeverityWarning},
				{Severity: sdk.SeverityWarning},
				{Severity: sdk.SeverityInfo},
				{Severity: sdk.SeverityInfo},
				{Severity: sdk.SeverityInfo},
			},
			wantErrors:   1,
			wantWarnings: 2,
			wantInfo:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, warnings, info := countBySeverity(tt.findings)
			assert.Equal(t, tt.wantErrors, errors)
			assert.Equal(t, tt.wantWarnings, warnings)
			assert.Equal(t, tt.wantInfo, info)
		})
	}
}
