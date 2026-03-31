package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
