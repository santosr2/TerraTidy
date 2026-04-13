package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unicode tests verify correct handling of non-ASCII characters in paths and patterns.
// These tests complement fuzz tests which check for no-panic but not correctness.

func TestMatchGlobPattern_UnicodePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		pattern  string
		expected bool
	}{
		// Japanese characters in path
		{
			name:     "japanese directory name with wildcard",
			filePath: "モジュール/main.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		{
			name:     "japanese directory exact match",
			filePath: "モジュール/vpc/main.tf",
			pattern:  "モジュール/**",
			expected: true,
		},
		{
			name:     "japanese filename",
			filePath: "modules/設定.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		// Chinese characters
		{
			name:     "chinese directory name",
			filePath: "模块/main.tf",
			pattern:  "模块/**",
			expected: true,
		},
		{
			name:     "chinese path with wildcard",
			filePath: "项目/模块/配置.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		// Korean characters
		{
			name:     "korean directory name",
			filePath: "모듈/main.tf",
			pattern:  "모듈/**",
			expected: true,
		},
		// Emoji in path (some filesystems support this)
		{
			name:     "emoji in directory name",
			filePath: "🚀-deploy/main.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		// Mixed ASCII and unicode
		{
			name:     "mixed ascii and unicode",
			filePath: "modules/日本語-module/main.tf",
			pattern:  "modules/**/*.tf",
			expected: true,
		},
		// Unicode with special glob characters
		{
			name:     "unicode directory with exact pattern",
			filePath: "テスト/main.tf",
			pattern:  "テスト/*.tf",
			expected: true,
		},
		// Non-matching cases
		{
			name:     "unicode path does not match different pattern",
			filePath: "モジュール/main.tf",
			pattern:  "modules/**",
			expected: false,
		},
		{
			name:     "unicode extension check",
			filePath: "modules/main.テスト",
			pattern:  "**/*.tf",
			expected: false,
		},
		// Accented latin characters
		{
			name:     "accented characters in path",
			filePath: "módulos/configuración/main.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		{
			name:     "accented directory exact match",
			filePath: "configuración/main.tf",
			pattern:  "configuración/**",
			expected: true,
		},
		// Cyrillic characters
		{
			name:     "cyrillic directory name",
			filePath: "модули/main.tf",
			pattern:  "модули/**",
			expected: true,
		},
		// Arabic characters (RTL)
		{
			name:     "arabic directory name",
			filePath: "وحدات/main.tf",
			pattern:  "وحدات/**",
			expected: true,
		},
		// Deep nesting with unicode
		{
			name:     "deep unicode path",
			filePath: "プロジェクト/モジュール/サブ/main.tf",
			pattern:  "**/*.tf",
			expected: true,
		},
		// Unicode prefix with ** mid-pattern
		{
			name:     "unicode prefix with doublestar suffix",
			filePath: "モジュール/sub/nested/main.tf",
			pattern:  "モジュール/**/main.tf",
			expected: true,
		},
		// Unicode filename with single-star pattern
		{
			name:     "unicode filename with single star",
			filePath: "modules/設定.tf",
			pattern:  "modules/*.tf",
			expected: true,
		},
		{
			name:     "unicode in both path and pattern single star",
			filePath: "モジュール/設定.tf",
			pattern:  "モジュール/*.tf",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchGlobPattern(tt.filePath, tt.pattern)
			assert.Equal(t, tt.expected, result, "path=%q pattern=%q", tt.filePath, tt.pattern)
		})
	}
}
