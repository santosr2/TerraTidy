package main

import (
	"testing"
)

// FuzzGlobPatternMatch tests the glob pattern matching functions with arbitrary
// file paths and patterns. Exercises matchGlobPattern and matchDoubleStarPattern
// to ensure no panics occur with edge-case input.
func FuzzGlobPatternMatch(f *testing.F) {
	// Standard Terraform file patterns
	f.Add("main.tf", "*.tf")
	f.Add("variables.tf", "*.tf")
	f.Add("path/to/main.tf", "**/*.tf")
	f.Add("modules/vpc/main.tf", "modules/**/*.tf")

	// Exact matches
	f.Add("main.tf", "main.tf")
	f.Add("path/to/file.tf", "path/to/file.tf")

	// Directory patterns
	f.Add("vendor/module/main.tf", "vendor/**")
	f.Add(".terraform/providers/main.tf", ".terraform/**")
	f.Add("path/vendor/nested/file.tf", "**/vendor/**")

	// Extension patterns
	f.Add("file.generated.tf", "*.generated.tf")
	f.Add("path/file.generated.tf", "**/*.generated.tf")
	f.Add("file.backup.tf", "*.backup.tf")

	// Single directory component matching
	f.Add("vendor/main.tf", "vendor")
	f.Add("path/to/vendor/main.tf", "vendor")
	f.Add("a/build-dir/c/main.tf", "build-*") // Wildcard against interior segment

	// Empty and minimal patterns
	f.Add("file.tf", "")
	f.Add("", "*.tf")
	f.Add("", "")

	// Patterns with multiple wildcards
	f.Add("a/b/c/d.tf", "a/**/c/*.tf")
	f.Add("deep/nested/path/file.tf", "**/path/**")
	f.Add("x/y/z.tf", "**/**/z.tf")

	// Special characters in paths
	f.Add("file-with-dashes.tf", "file-*.tf")
	f.Add("file_with_underscores.tf", "file_*.tf")
	f.Add("file.with.dots.tf", "file.*.tf")
	f.Add("path with spaces/file.tf", "**/*.tf")

	// Unicode paths
	f.Add("日本語/main.tf", "**/*.tf")
	f.Add("файл.tf", "*.tf")
	f.Add("αρχείο/nested/file.tf", "αρχείο/**")

	// Deeply nested paths
	f.Add("a/b/c/d/e/f/g/h/i/j/k/l/main.tf", "**/*.tf")
	f.Add("level1/level2/level3/level4/level5/file.tf", "level1/**/file.tf")

	// Patterns that shouldn't match
	f.Add("main.go", "*.tf")
	f.Add("src/main.go", "**/*.tf")
	f.Add("other/path/file.txt", "vendor/**")

	// Edge cases with slashes
	f.Add("/absolute/path/file.tf", "**/*.tf")
	f.Add("./relative/path/file.tf", "**/*.tf")
	f.Add("path//double//slash/file.tf", "**/*.tf")
	f.Add("trailing/slash/", "trailing/**")

	// Bracket and brace patterns (may or may not be supported)
	f.Add("file1.tf", "file[0-9].tf")
	f.Add("file.tf", "[f]ile.tf")

	// Question mark patterns
	f.Add("main.tf", "mai?.tf")
	f.Add("file1.tf", "file?.tf")

	// Very long patterns
	longPath := ""
	for i := range 50 {
		longPath += "dir" + string(rune('a'+i%26)) + "/"
	}
	longPath += "file.tf"
	f.Add(longPath, "**/*.tf")
	f.Add(longPath, "dira/**/file.tf")

	// Very long pattern with many wildcards
	f.Add("a/b/c/d.tf", "**/**/**/**/*.tf")
	f.Add("single.tf", "**/**/**/**/*.tf")

	// Binary-looking paths (shouldn't panic)
	f.Add(string([]byte{0x00, 0x01})+".tf", "*.tf")
	f.Add("file\x00null.tf", "**/*.tf")

	f.Fuzz(func(t *testing.T, filePath, pattern string) {
		// matchGlobPattern should never panic
		// (correctness assertions are in unit tests; fuzz tests catch panics)
		_ = matchGlobPattern(filePath, pattern)
	})
}

// FuzzMatchesAnyPattern tests the matchesAnyPattern function with arbitrary
// file paths and multiple patterns.
func FuzzMatchesAnyPattern(f *testing.F) {
	// Single pattern cases
	f.Add("main.tf", "*.tf", "")
	f.Add("vendor/file.tf", "vendor/**", "")

	// Multiple patterns
	f.Add("main.tf", "*.tf", "*.go")
	f.Add("vendor/file.tf", "vendor/**", ".terraform/**")
	f.Add("file.generated.tf", "*.generated.tf", "*.backup.tf")

	// Empty patterns
	f.Add("file.tf", "", "")
	f.Add("", "*.tf", "*.go")

	// Mix of matching and non-matching
	f.Add("src/main.go", "*.tf", "**/*.go")
	f.Add("vendor/nested/deep/file.tf", "vendor/**", "node_modules/**")

	// Unicode and special chars
	f.Add("日本語.tf", "*.tf", "*.go")
	f.Add("file-name.tf", "file-*.tf", "other-*.tf")

	// Note: fuzz target signature is fixed at 3 strings (filePath, pattern1, pattern2).
	// Multi-pattern stress testing is done by the fuzzer varying both pattern args.

	f.Fuzz(func(t *testing.T, filePath, pattern1, pattern2 string) {
		// Pass patterns through unsanitized - empty strings are valid edge cases
		// (filepath.Match("", "file.tf") returns false, nil)
		// Correctness assertions are in unit tests; fuzz tests catch panics
		_ = matchesAnyPattern(filePath, []string{pattern1, pattern2})
	})
}

// FuzzDoubleStarPattern tests matchDoubleStarPattern directly, bypassing
// matchGlobPattern's routing. This exercises the len(parts)==1 fallback branch
// (when pattern contains no **) which matchGlobPattern routes elsewhere.
// Some seeds overlap with FuzzGlobPatternMatch intentionally for direct coverage.
func FuzzDoubleStarPattern(f *testing.F) {
	// Basic ** patterns
	f.Add("path/to/file.tf", "**/*.tf")
	f.Add("file.tf", "**/*.tf")
	f.Add("a/b/c/d/file.tf", "**/*.tf")

	// Prefix patterns
	f.Add("vendor/file.tf", "vendor/**")
	f.Add("vendor/nested/file.tf", "vendor/**")
	f.Add("other/file.tf", "vendor/**")

	// Suffix patterns
	f.Add("path/to/main.tf", "**/*.tf")
	f.Add("path/to/main.go", "**/*.tf")

	// Middle ** patterns
	f.Add("src/vendor/file.tf", "**/vendor/**")
	f.Add("a/b/vendor/c/d.tf", "**/vendor/**")
	f.Add("vendor/file.tf", "**/vendor/**")

	// Multiple ** segments
	f.Add("a/b/c/d/e.tf", "a/**/c/**/e.tf")
	f.Add("a/c/e.tf", "a/**/c/**/e.tf")

	// ** at start and end
	f.Add("anything/goes/here", "**")
	f.Add("single", "**")
	f.Add("", "**")

	// Edge cases
	f.Add("path//double/file.tf", "**/file.tf")
	f.Add("./relative.tf", "**/*.tf")
	f.Add("/absolute/path.tf", "**/*.tf")

	// Patterns without ** (fallback path)
	f.Add("main.tf", "main.tf")
	f.Add("main.tf", "*.tf")
	f.Add("path/file.tf", "path/file.tf")

	// Unicode in paths
	f.Add("日本語/ファイル.tf", "**/*.tf")
	f.Add("путь/файл.tf", "путь/**")

	// Stress test with many path segments
	deepPath := ""
	for range 100 {
		deepPath += "dir/"
	}
	deepPath += "file.tf"
	f.Add(deepPath, "**/*.tf")
	f.Add(deepPath, "dir/**/file.tf")

	f.Fuzz(func(t *testing.T, filePath, pattern string) {
		// matchDoubleStarPattern should never panic
		// Correctness assertions are in unit tests; fuzz tests catch panics
		// Seeds under "Patterns without **" exercise the len(parts)==1 fallback
		_ = matchDoubleStarPattern(filePath, pattern)
	})
}
