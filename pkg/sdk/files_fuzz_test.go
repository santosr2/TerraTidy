package sdk

import (
	"path/filepath"
	"strings"
	"testing"
)

// FuzzIsHCLFile tests the IsHCLFile function with arbitrary file paths.
// It exercises the extension detection logic and verifies the invariant that
// the function returns true only for .tf, .hcl, or .tfvars extensions.
func FuzzIsHCLFile(f *testing.F) {
	// Valid HCL file paths
	f.Add("main.tf")
	f.Add("variables.tf")
	f.Add("outputs.tf")
	f.Add("terraform.tfvars")
	f.Add("config.hcl")
	f.Add("settings.hcl")

	// With directories
	f.Add("modules/service/main.tf")
	f.Add("environments/dev.tfvars")
	f.Add("a/b/c/d/e/f/g/main.tf")

	// Case variations (extension detection uses ToLower)
	f.Add("main.TF")
	f.Add("main.Tf")
	f.Add("main.tF")
	f.Add("config.HCL")
	f.Add("terraform.TFVARS")
	f.Add("main.TfVaRs")

	// Non-HCL files
	f.Add("README.md")
	f.Add("main.go")
	f.Add("package.json")
	f.Add("Makefile")
	f.Add("Dockerfile")
	f.Add(".gitignore")
	f.Add(".terraform.lock.hcl") // Actually HCL
	f.Add("terraform.tfvars.json")

	// Edge cases: no extension
	f.Add("terraform")
	f.Add("LICENSE")

	// Empty and whitespace
	f.Add("")
	f.Add(" ")
	f.Add("   ")
	f.Add("\t")
	f.Add("\n")
	f.Add("\r\n")

	// Special characters in paths
	f.Add("path with spaces/main.tf")
	f.Add("path-with-dashes/main.tf")
	f.Add("path_with_underscores/main.tf")
	f.Add("path.with.dots/main.tf")
	f.Add("path/to/file.with.multiple.dots.tf")

	// Windows-style paths
	f.Add(`C:\terraform\main.tf`)
	f.Add(`C:\Users\dev\project\variables.tf`)
	f.Add(`\\server\share\terraform\main.tf`)

	// Unicode paths
	f.Add("モジュール/main.tf")
	f.Add("مجلد/main.tf")
	f.Add("папка/main.tf")
	f.Add("目录/main.tf")
	f.Add("ディレクトリ/設定.tf")

	// Very long paths
	f.Add("a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z/main.tf")
	longPath := make([]byte, 1000)
	for i := range longPath {
		longPath[i] = 'a'
	}
	f.Add(string(longPath) + ".tf")

	// Special file extensions
	f.Add("file.tf.bak")
	f.Add("file.tf.backup")
	f.Add("file.tf.orig")
	f.Add(".tf")
	f.Add("..tf")
	f.Add("...tf")

	// Multiple extensions
	f.Add("archive.tar.gz")
	f.Add("backup.tf.gz")
	f.Add("main.tf.tf")

	// Hidden files
	f.Add(".hidden.tf")
	f.Add(".terraform/main.tf")
	f.Add("..parent/main.tf")

	// Null bytes and binary
	f.Add("file\x00.tf")
	f.Add("\x00\x01\x02")

	// Path traversal patterns
	f.Add("../main.tf")
	f.Add("../../main.tf")
	f.Add("./main.tf")
	f.Add("modules/../main.tf")

	f.Fuzz(func(t *testing.T, path string) {
		result := IsHCLFile(path)

		// Verify invariant: result should match independent extension check.
		// This catches any regressions in extension handling.
		ext := strings.ToLower(filepath.Ext(path))
		want := ext == ".tf" || ext == ".hcl" || ext == ".tfvars"
		if result != want {
			t.Errorf("IsHCLFile(%q) = %v, want %v (ext=%q)", path, result, want, ext)
		}
	})
}

// FuzzGroupFilesByDirectory tests the GroupFilesByDirectory function with arbitrary file paths.
// It exercises the directory grouping logic and verifies that all input files appear
// in the output exactly once, grouped by their parent directory.
func FuzzGroupFilesByDirectory(f *testing.F) {
	// Simple cases
	f.Add([]byte("main.tf"))
	f.Add([]byte("modules/main.tf"))
	f.Add([]byte("a/b/c/main.tf"))

	// Multiple files (newline separated)
	f.Add([]byte("main.tf\nvariables.tf\noutputs.tf"))
	f.Add([]byte("modules/a/main.tf\nmodules/b/main.tf"))

	// Duplicate paths (tests append behavior)
	f.Add([]byte("main.tf\n" + "main.tf"))
	f.Add([]byte("modules/main.tf\n" + "modules/main.tf\n" + "modules/main.tf"))

	// Empty and whitespace
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("   "))
	f.Add([]byte("\t\t\t"))

	// Special characters
	f.Add([]byte("path with spaces/main.tf"))
	f.Add([]byte("path-with-dashes/main.tf"))
	f.Add([]byte("path.with.dots/main.tf"))

	// Windows paths
	f.Add([]byte(`C:\terraform\main.tf`))
	f.Add([]byte(`\\server\share\main.tf`))

	// Unicode
	f.Add([]byte("モジュール/main.tf"))
	f.Add([]byte("目录/文件.tf"))

	// Deep nesting
	f.Add([]byte("a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/main.tf"))

	// Many files in same directory
	manyFiles := make([]byte, 0, 1000)
	for i := range 50 {
		if i > 0 {
			manyFiles = append(manyFiles, '\n')
		}
		manyFiles = append(manyFiles, []byte("modules/service/file"+string(rune('0'+i%10))+".tf")...)
	}
	f.Add(manyFiles)

	// Many directories
	manyDirs := make([]byte, 0, 1000)
	for i := range 50 {
		if i > 0 {
			manyDirs = append(manyDirs, '\n')
		}
		manyDirs = append(manyDirs, []byte("modules/service-"+string(rune('0'+i%10))+"/main.tf")...)
	}
	f.Add(manyDirs)

	// Path edge cases
	f.Add([]byte("."))
	f.Add([]byte(".."))
	f.Add([]byte("./"))
	f.Add([]byte("../"))
	f.Add([]byte("/"))
	f.Add([]byte("//"))

	// Relative vs absolute
	f.Add([]byte("/absolute/path/main.tf"))
	f.Add([]byte("relative/path/main.tf"))
	f.Add([]byte("./relative/path/main.tf"))
	f.Add([]byte("../parent/path/main.tf"))

	// Binary data
	f.Add([]byte{0x00, 0x01, 0x02})
	f.Add([]byte("file\x00path"))

	// Long paths
	longPath := make([]byte, 500)
	for i := range longPath {
		if i > 0 && i%10 == 0 {
			longPath[i] = '/'
		} else {
			longPath[i] = 'a'
		}
	}
	f.Add(longPath)

	f.Fuzz(func(t *testing.T, data []byte) {
		paths := parsePathsFromBytes(data)
		result := GroupFilesByDirectory(paths)

		// Verify invariant: every input path must appear in the output exactly once,
		// and each file must be grouped under its correct parent directory.
		total := 0
		for dir, files := range result {
			for _, file := range files {
				if filepath.Dir(file) != dir {
					t.Errorf("file %q placed in wrong directory %q (expected %q)", file, dir, filepath.Dir(file))
				}
				total++
			}
		}
		if total != len(paths) {
			t.Errorf("GroupFilesByDirectory returned %d files for %d inputs", total, len(paths))
		}
	})
}

// parsePathsFromBytes converts fuzz input bytes into a slice of file paths.
// Uses newline as separator to generate multiple paths from single input.
//
// Design notes:
//   - Empty segments (consecutive newlines) are skipped
//   - Whitespace-only segments are kept deliberately to exercise edge cases
//   - CR characters (\r) are kept as part of paths to test Windows line endings
func parsePathsFromBytes(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	var paths []string
	var current []byte

	for _, b := range data {
		if b == '\n' {
			if len(current) > 0 {
				paths = append(paths, string(current))
				current = nil
			}
		} else {
			current = append(current, b)
		}
	}

	// Don't forget the last path (no trailing newline)
	if len(current) > 0 {
		paths = append(paths, string(current))
	}

	return paths
}
