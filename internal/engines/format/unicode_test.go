package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unicode content tests verify that HCL formatting preserves unicode characters.
// These tests complement fuzz tests which check for no-panic but not correctness.

func TestFormat_UnicodeContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Japanese characters in strings
		{
			name: "japanese string value",
			input: `variable "message" {
  default = "こんにちは世界"
}
`,
			want: `variable "message" {
  default = "こんにちは世界"
}
`,
		},
		// Chinese characters
		{
			name: "chinese string value",
			input: `variable "greeting" {
  default = "你好世界"
}
`,
			want: `variable "greeting" {
  default = "你好世界"
}
`,
		},
		// Korean characters
		{
			name: "korean string value",
			input: `variable "message" {
  default = "안녕하세요"
}
`,
			want: `variable "message" {
  default = "안녕하세요"
}
`,
		},
		// Emoji in strings
		{
			name: "emoji in string",
			input: `variable "status" {
  default = "🚀 Deployed successfully ✅"
}
`,
			want: `variable "status" {
  default = "🚀 Deployed successfully ✅"
}
`,
		},
		// Arabic characters (RTL)
		{
			name: "arabic string value",
			input: `variable "message" {
  default = "مرحبا بالعالم"
}
`,
			want: `variable "message" {
  default = "مرحبا بالعالم"
}
`,
		},
		// Cyrillic characters
		{
			name: "cyrillic string value",
			input: `variable "message" {
  default = "Привет мир"
}
`,
			want: `variable "message" {
  default = "Привет мир"
}
`,
		},
		// Accented latin characters
		{
			name: "accented latin characters",
			input: `variable "description" {
  default = "Configuración del módulo"
}
`,
			want: `variable "description" {
  default = "Configuración del módulo"
}
`,
		},
		// Unicode in comments
		{
			name: "unicode in comment",
			input: `# 日本語のコメント
variable "x" {
  default = "value"
}
`,
			want: `# 日本語のコメント
variable "x" {
  default = "value"
}
`,
		},
		// Unicode in resource names (identifiers)
		{
			name: "unicode variable name",
			input: `variable "日本語" {
  default = "value"
}
`,
			want: `variable "日本語" {
  default = "value"
}
`,
		},
		// Mixed unicode and ASCII
		{
			name: "mixed unicode and ascii",
			input: `variable "mixed" {
  default = "Hello 世界 - Привет мир - مرحبا"
}
`,
			want: `variable "mixed" {
  default = "Hello 世界 - Привет мир - مرحبا"
}
`,
		},
		// Unicode with formatting needed
		{
			name: "unicode content needs formatting",
			input: `variable "message"   {
default="こんにちは"
}
`,
			want: `variable "message" {
  default = "こんにちは"
}
`,
		},
		// Unicode in heredoc
		{
			name: "unicode in heredoc",
			input: `variable "script" {
  default = <<-EOT
    echo "こんにちは世界"
    echo "Привет мир"
  EOT
}
`,
			want: `variable "script" {
  default = <<-EOT
    echo "こんにちは世界"
    echo "Привет мир"
  EOT
}
`,
		},
		// Unicode in tags map
		{
			name: "unicode in tags",
			input: `resource "aws_instance" "example" {
  tags = {
    Name        = "サーバー"
    Environment = "本番環境"
  }
}
`,
			want: `resource "aws_instance" "example" {
  tags = {
    Name        = "サーバー"
    Environment = "本番環境"
  }
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format([]byte(tt.input))
			assert.Equal(t, tt.want, string(got), "unicode content should be preserved after formatting")
		})
	}
}
