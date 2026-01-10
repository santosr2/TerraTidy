package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHCLFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.tf", true},
		{"MAIN.TF", true},
		{"variables.tf", true},
		{"config.hcl", true},
		{"terraform.tfvars", true},
		{"main.go", false},
		{"README.md", false},
		{"config.json", false},
		{"module/main.tf", true},
		{"path/to/file.hcl", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isHCLFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{"normal directory", "modules", false},
		{"hidden directory", ".git", true},
		{"terraform cache", ".terraform", true},
		{"terragrunt cache", ".terragrunt-cache", true},
		{"node_modules", "node_modules", true},
		{"vendor", "vendor", true},
		{"pycache", "__pycache__", true},
		{"current dir", ".", false},
		{"src directory", "src", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipDir("", tt.dirName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPathWithin(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		dirPath  string
		expected bool
	}{
		{
			name:     "file directly in directory",
			filePath: "/project/main.tf",
			dirPath:  "/project",
			expected: true,
		},
		{
			name:     "file in subdirectory",
			filePath: "/project/modules/vpc/main.tf",
			dirPath:  "/project/modules",
			expected: true,
		},
		{
			name:     "file outside directory",
			filePath: "/other/main.tf",
			dirPath:  "/project",
			expected: false,
		},
		{
			name:     "prefix mismatch",
			filePath: "/project-other/main.tf",
			dirPath:  "/project",
			expected: false,
		},
		{
			name:     "exact match",
			filePath: "/project/main.tf",
			dirPath:  "/project/main.tf",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathWithin(tt.filePath, tt.dirPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFileCount(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "0 files"},
		{1, "1 file"},
		{2, "2 files"},
		{10, "10 files"},
		{100, "100 files"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileCount(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToAbsPath(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		result := toAbsPath("main.tf")
		assert.True(t, filepath.IsAbs(result))
	})

	t.Run("absolute path", func(t *testing.T) {
		absPath := "/absolute/path/main.tf"
		result := toAbsPath(absPath)
		assert.Equal(t, absPath, result)
	})
}

func TestFindHCLFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte("# test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# readme"), 0o644))

	// Create subdirectory with terraform files
	subDir := filepath.Join(tmpDir, "modules", "vpc")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "main.tf"), []byte("# vpc"), 0o644))

	// Create hidden directory that should be skipped
	hiddenDir := filepath.Join(tmpDir, ".terraform")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "cached.tf"), []byte("# cache"), 0o644))

	t.Run("finds all HCL files", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir})
		require.NoError(t, err)
		assert.Len(t, files, 3) // main.tf, variables.tf, modules/vpc/main.tf
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir})
		require.NoError(t, err)

		for _, f := range files {
			assert.NotContains(t, f, ".terraform")
		}
	})

	t.Run("handles single file path", func(t *testing.T) {
		singleFile := filepath.Join(tmpDir, "main.tf")
		files, err := findHCLFiles([]string{singleFile})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("handles non-existent path", func(t *testing.T) {
		_, err := findHCLFiles([]string{"/non/existent/path"})
		assert.Error(t, err)
	})
}

func TestFindHCLFilesFromPaths(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.tf"), []byte("# test"), 0o644))

	t.Run("uses current directory when empty", func(t *testing.T) {
		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		files, err := findHCLFilesFromPaths([]string{})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("uses provided paths", func(t *testing.T) {
		files, err := findHCLFilesFromPaths([]string{tmpDir})
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})
}

func TestFileCollector(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))

	t.Run("collects unique files", func(t *testing.T) {
		collector := newFileCollector()
		err := collector.collectPath(tmpDir)
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)

		// Collect same path again - should not duplicate
		err = collector.collectPath(tmpDir)
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)
	})

	t.Run("handles single file", func(t *testing.T) {
		collector := newFileCollector()
		err := collector.collectPath(filepath.Join(tmpDir, "main.tf"))
		require.NoError(t, err)
		assert.Len(t, collector.files, 1)
	})
}

func TestGetTargetFiles(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("# test"), 0o644))

	t.Run("without changed flag", func(t *testing.T) {
		files, err := getTargetFiles([]string{tmpDir}, false)
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("with changed flag outside git repo", func(t *testing.T) {
		// Create a non-git directory
		nonGitDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(nonGitDir, "main.tf"), []byte("# test"), 0o644))

		oldWd, _ := os.Getwd()
		_ = os.Chdir(nonGitDir)
		defer func() { _ = os.Chdir(oldWd) }()

		_, err := getTargetFiles([]string{nonGitDir}, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}
