//go:build windows

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathsEqual_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"same case", `C:\project\main.tf`, `C:\project\main.tf`, true},
		{"different drive letter case", `C:\project\main.tf`, `c:\project\main.tf`, true},
		{"different path case", `C:\PROJECT\main.tf`, `C:\project\MAIN.TF`, true},
		{"mixed case match", `C:\Users\Dev\Main.TF`, `c:\users\dev\main.tf`, true},
		{"different paths still different", `C:\project\main.tf`, `D:\project\main.tf`, false},
		{"empty paths", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "pathsEqual(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestHasPathPrefix_WindowsDrive(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected bool
	}{
		{"same drive", `C:\project\main.tf`, `C:\project`, true},
		{"case insensitive drive", `C:\project\main.tf`, `c:\project`, true},
		{"case insensitive path", `C:\PROJECT\main.tf`, `c:\project`, true},
		{"different drives", `D:\project\main.tf`, `C:\project`, false},
		{"root prefix", `C:\project\main.tf`, `C:\`, true},
		// Note: hasPathPrefix is a raw prefix check, not a path-boundary check
		{"partial prefix mismatch still matches", `C:\project-other\main.tf`, `C:\project`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPathPrefix(tt.path, tt.prefix)
			assert.Equal(t, tt.expected, result, "hasPathPrefix(%q, %q)", tt.path, tt.prefix)
		})
	}
}

func TestIsPathWithin_WindowsCrossDrive(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		dirPath  string
		expected bool
	}{
		{"same drive within", `C:\project\modules\main.tf`, `C:\project`, true},
		{"same drive directly in", `C:\project\main.tf`, `C:\project`, true},
		{"different drives", `D:\project\main.tf`, `C:\project`, false},
		{"case insensitive within", `C:\PROJECT\main.tf`, `c:\project`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathWithin(tt.filePath, tt.dirPath)
			assert.Equal(t, tt.expected, result, "isPathWithin(%q, %q)", tt.filePath, tt.dirPath)
		})
	}
}
