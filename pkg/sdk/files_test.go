package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupFilesByDirectory(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  map[string][]string
	}{
		{"nil", nil, map[string][]string{}},
		{"empty", []string{}, map[string][]string{}},
		{"single file", []string{"/a/b/main.tf"}, map[string][]string{"/a/b": {"/a/b/main.tf"}}},
		{"same dir", []string{"/a/main.tf", "/a/vars.tf"}, map[string][]string{"/a": {"/a/main.tf", "/a/vars.tf"}}},
		{"different dirs", []string{"/a/main.tf", "/b/main.tf"}, map[string][]string{
			"/a": {"/a/main.tf"},
			"/b": {"/b/main.tf"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupFilesByDirectory(tt.files)
			assert.Equal(t, tt.want, got)
		})
	}
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
