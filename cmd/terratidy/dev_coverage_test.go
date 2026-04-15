package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRelevantFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"rego file", "policy.rego", true},
		{"tf file", "main.tf", true},
		{"hcl file", "config.hcl", true},
		{"tfvars file", "dev.tfvars", true},
		{"go file", "main.go", false},
		{"yaml file", "config.yaml", false},
		{"no extension", "README", false},
		{"json file", "data.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRelevantFile(tt.file))
		})
	}
}

func TestFindPolicyFiles(t *testing.T) {
	dir := t.TempDir()

	// Create rego files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy.rego"), []byte("package main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy_test.rego"), []byte("package test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.rego"), []byte("package other"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-rego.tf"), []byte("resource {}"), 0o644))

	files, err := findPolicyFiles(dir)
	require.NoError(t, err)

	// Should include .rego files but exclude _test.rego
	assert.Len(t, files, 2)
	for _, f := range files {
		assert.NotContains(t, f, "_test.rego", "should exclude test files")
	}
}

func TestFindPolicyFiles_NonExistentDir(t *testing.T) {
	_, err := findPolicyFiles("/nonexistent/dir")
	assert.Error(t, err)
}

func TestFindPolicyFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, err := findPolicyFiles(dir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestWatchDirExists(t *testing.T) {
	t.Run("existing directory", func(t *testing.T) {
		dir := t.TempDir()
		oldDevWatch := devWatch
		devWatch = dir
		defer func() { devWatch = oldDevWatch }()

		assert.True(t, watchDirExists())
	})

	t.Run("non-existing directory", func(t *testing.T) {
		oldDevWatch := devWatch
		devWatch = "/nonexistent/path"
		defer func() { devWatch = oldDevWatch }()

		assert.False(t, watchDirExists())
	})
}

func TestPrintWatchDirMissingHelp_QuotesPathWithSpecialChars(t *testing.T) {
	// Save and restore devWatch
	oldDevWatch := devWatch
	defer func() { devWatch = oldDevWatch }()

	// Test path with spaces and special characters
	devWatch = "/path/with spaces/and'quotes"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printWatchDirMissingHelp()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// The mkdir hint should use quoted path (via %q) to handle special chars
	// %q produces: "/path/with spaces/and'quotes"
	assert.Contains(t, output, `mkdir -p "/path/with spaces/and'quotes"`,
		"path with special chars should be properly quoted in shell hint")
}

func TestDevSeverityIcon(t *testing.T) {
	tests := []struct {
		name     string
		severity sdk.Severity
		want     string
	}{
		{"error severity", sdk.SeverityError, "E"},
		{"warning severity", sdk.SeverityWarning, "W"},
		{"info severity", sdk.SeverityInfo, "i"},
		{"unknown severity returns ?", sdk.Severity("unknown"), "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := devSeverityIcon(tt.severity)
			assert.Equal(t, tt.want, got)
		})
	}
}
