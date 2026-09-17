package hcltest

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nested = `region = "eu-west-1"

resource "aws_instance" "web" {
  ami = "ami-123"
  lifecycle {
    create_before_destroy = true
  }
}
`

func parseBody(t *testing.T, src string) *hclsyntax.Body {
	t.Helper()
	file, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	require.False(t, diags.HasErrors(), diags.Error())
	return file.Body.(*hclsyntax.Body)
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"valid HCL", []byte(nested), true},
		{"empty", nil, true},
		{"syntax error", []byte("a = "), false},
		{"invalid UTF-8", []byte("a = \"\xc9\"\n"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValid(tt.input))
		})
	}
}

func TestTopLevelNames(t *testing.T) {
	assert.Equal(t, map[string]bool{
		"attr:region":                     true,
		"block:resource:aws_instance:web": true,
	}, TopLevelNames(parseBody(t, nested)))
}

func TestAllNames(t *testing.T) {
	assert.Equal(t, map[string]bool{
		"attr:region":                                       true,
		"block:resource:aws_instance:web":                   true,
		"block:resource:aws_instance:web > attr:ami":        true,
		"block:resource:aws_instance:web > block:lifecycle": true,
		"block:resource:aws_instance:web > block:lifecycle > attr:create_before_destroy": true,
	}, AllNames(parseBody(t, nested)))
}

func TestAllNamesDistinguishesEnclosingBlock(t *testing.T) {
	inBlock := AllNames(parseBody(t, "a {\n  x = 1\n}\nb {}\n"))
	moved := AllNames(parseBody(t, "a {}\nb {\n  x = 1\n}\n"))
	assert.NotEqual(t, inBlock, moved)
}
