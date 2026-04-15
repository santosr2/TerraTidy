package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountFormattedFiles(t *testing.T) {
	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"empty", []sdk.Finding{}, 0},
		{"no formatted", []sdk.Finding{{Rule: "style.something"}}, 0},
		{"one formatted", []sdk.Finding{{Rule: "fmt.formatted"}}, 1},
		{"mixed", []sdk.Finding{
			{Rule: "fmt.formatted"},
			{Rule: "style.blank-line"},
			{Rule: "fmt.formatted"},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countFormattedFiles(tt.findings))
		})
	}
}

func TestCountFixedStyleIssues(t *testing.T) {
	fixResult := &sdk.FixResult{Content: []byte("fixed")}

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"not fixable", []sdk.Finding{{Fix: nil}}, 0},
		{"fixable with fix", []sdk.Finding{{Fix: fixResult}}, 1},
		{"mixed", []sdk.Finding{
			{Fix: fixResult},
			{Fix: nil},
			{Fix: fixResult},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countFixedStyleIssues(tt.findings))
		})
	}
}

func TestCountRemainingIssues(t *testing.T) {
	fixResult := &sdk.FixResult{Content: []byte("fixed")}

	tests := []struct {
		name     string
		findings []sdk.Finding
		want     int
	}{
		{"nil", nil, 0},
		{"all fixable", []sdk.Finding{{Fix: fixResult}}, 0},
		{"not fixable", []sdk.Finding{{Fix: nil}}, 1},
		{"mixed", []sdk.Finding{
			{Fix: fixResult},
			{Fix: nil},
			{Fix: nil},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countRemainingIssues(tt.findings))
		})
	}
}

func TestRunFix_ErrorPaths(t *testing.T) {
	t.Run("invalid config returns ExitConfig", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create invalid config
		invalidConfig := "invalid: yaml: ["
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".terratidy.yaml"), []byte(invalidConfig), 0o600))

		oldWd, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		// Reset global flags
		cfgFile = ""
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"fix", "."})
		err := rootCmd.Execute()
		require.Error(t, err)

		var exitErr *sdk.ExitError
		if errors.As(err, &exitErr) {
			assert.Equal(t, sdk.ExitConfig, exitErr.Code, "invalid config should return ExitConfig")
		}
	})

	t.Run("no files found", func(t *testing.T) {
		emptyDir := t.TempDir()
		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"fix", emptyDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("fix valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `resource "null_resource" "test" {
  triggers = {
    a = "b"
  }
}
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

		changed = false
		format = "text"

		rootCmd.SetArgs([]string{"fix", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("fix with structured output", func(t *testing.T) {
		tmpDir := t.TempDir()
		content := `resource "null_resource" "test" {}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(content), 0o644))

		changed = false
		format = "json"

		rootCmd.SetArgs([]string{"fix", tmpDir})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})
}
