package sdk

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupFilesByDirectory(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got := GroupFilesByDirectory(nil)
		assert.Empty(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		got := GroupFilesByDirectory([]string{})
		assert.Empty(t, got)
	})

	t.Run("single file", func(t *testing.T) {
		f := filepath.Join("a", "b", "main.tf")
		got := GroupFilesByDirectory([]string{f})
		dir := filepath.Join("a", "b")
		require.Contains(t, got, dir)
		assert.Equal(t, []string{f}, got[dir])
	})

	t.Run("same dir", func(t *testing.T) {
		f1 := filepath.Join("a", "main.tf")
		f2 := filepath.Join("a", "vars.tf")
		got := GroupFilesByDirectory([]string{f1, f2})
		dir := filepath.Join("a")
		require.Contains(t, got, dir)
		assert.Equal(t, []string{f1, f2}, got[dir])
	})

	t.Run("different dirs", func(t *testing.T) {
		f1 := filepath.Join("a", "main.tf")
		f2 := filepath.Join("b", "main.tf")
		got := GroupFilesByDirectory([]string{f1, f2})
		assert.Len(t, got, 2)
		assert.Contains(t, got, "a")
		assert.Contains(t, got, "b")
	})
}

func TestIsHCLFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.tf", true},
		{"variables.tf", true},
		{"outputs.hcl", true},
		{"terraform.tfvars", true},
		{"dev.auto.tfvars", true},
		{"/abs/path/main.tf", true},
		{"module/main.TF", true},
		{"main.go", false},
		{"config.yaml", false},
		{"README.md", false},
		{".tf", true},
		{"", false},
		{"noext", false},
		{"main.tf.bak", false},
		{"main.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsHCLFile(tt.path))
		})
	}
}
