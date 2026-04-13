package vcs

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
)

// FuzzParseFileList tests the file list parsing logic from git diff --name-only output.
// It exercises bufio.Scanner on arbitrary newline-separated content and ensures no panics.
//
// NOTE: This tests the core scanning logic from parseFileList (git.go:297-314) but NOT the
// path absolutization branch (filepath.Join with repoRoot) which requires a real git repository.
// The path joining is a standard Go operation unlikely to panic; the scanner loop is the
// interesting fuzzing target.
func FuzzParseFileList(f *testing.F) {
	// Valid file lists
	f.Add([]byte("main.tf\n"))
	f.Add([]byte("main.tf\nvariables.tf\n"))
	f.Add([]byte("main.tf\nvariables.tf\nmodules/vpc/main.tf\n"))
	f.Add([]byte("modules/vpc/main.tf\nmodules/rds/main.tf\n"))

	// Empty and whitespace
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("   \n   \n"))
	f.Add([]byte("\t\t\n"))

	// Paths with spaces
	f.Add([]byte("path with spaces/file.tf\n"))
	f.Add([]byte("  leading-space.tf\n"))
	f.Add([]byte("trailing-space.tf  \n"))

	// Paths with special characters
	f.Add([]byte("file-with-dashes.tf\n"))
	f.Add([]byte("file_with_underscores.tf\n"))
	f.Add([]byte("file.with.dots.tf\n"))
	f.Add([]byte("file@special.tf\n"))
	f.Add([]byte("file#hash.tf\n"))
	f.Add([]byte("file%percent.tf\n"))

	// Unicode paths
	f.Add([]byte("日本語/ファイル.tf\n"))
	f.Add([]byte("中文路径/文件.tf\n"))
	f.Add([]byte("путь/файл.tf\n"))
	f.Add([]byte("مسار/ملف.tf\n"))
	f.Add([]byte("émoji🎉/file.tf\n"))

	// Windows-style paths (git often outputs forward slashes)
	f.Add([]byte("C:/Users/test/main.tf\n"))
	f.Add([]byte("C:\\Users\\test\\main.tf\n"))

	// Mixed line endings
	f.Add([]byte("file1.tf\r\nfile2.tf\r\n"))
	f.Add([]byte("file1.tf\rfile2.tf\r"))
	f.Add([]byte("file1.tf\nfile2.tf\r\nfile3.tf\n"))

	// Very long paths
	f.Add([]byte("very/long/path/that/goes/on/and/on/and/on/and/on/and/on/file.tf\n"))
	f.Add([]byte(strings.Repeat("a/", 50) + "file.tf\n"))

	// Many files
	f.Add([]byte("a.tf\nb.tf\nc.tf\nd.tf\ne.tf\nf.tf\ng.tf\nh.tf\ni.tf\nj.tf\n"))

	// Quoted paths (git may quote paths with special chars)
	f.Add([]byte("\"quoted path/file.tf\"\n"))
	f.Add([]byte("'single quoted/file.tf'\n"))

	// Null bytes and binary
	f.Add([]byte("file\x00.tf\n"))
	f.Add([]byte{0x00, 0x01, 0x02, 0x0a})

	// Empty lines mixed with content
	f.Add([]byte("file1.tf\n\nfile2.tf\n\n\nfile3.tf\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Exercise the same parsing logic as parseFileList (git.go:297-314)
		// without the GetRepoRoot() call which requires a real git repo.
		var files []string
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				files = append(files, line)
			}
		}
		// scanner.Err() should not panic even with malformed input
		_ = scanner.Err()

		// Dereference results to catch nil/empty element panics
		for i, f := range files {
			_ = len(f)
			_ = i
		}
	})
}

// FuzzParseGitStatus tests the porcelain status parsing from git status --porcelain output.
// It exercises the status code extraction and path parsing using the real FileStatus type.
//
// NOTE: This tests the parsing logic from GetFileStatuses (git.go:341-365) but NOT the
// git command execution. The parsing loop is the interesting fuzzing target.
func FuzzParseGitStatus(f *testing.F) {
	// Valid porcelain status output
	f.Add([]byte(" M main.tf\n"))
	f.Add([]byte("M  main.tf\n"))
	f.Add([]byte("MM main.tf\n"))
	f.Add([]byte("A  new.tf\n"))
	f.Add([]byte("D  deleted.tf\n"))
	f.Add([]byte("R  old.tf -> new.tf\n"))
	f.Add([]byte("C  original.tf -> copy.tf\n"))
	f.Add([]byte("?? untracked.tf\n"))
	f.Add([]byte("!! ignored.tf\n"))

	// Multiple statuses
	f.Add([]byte(" M main.tf\nA  new.tf\n?? untracked.tf\n"))
	f.Add([]byte("M  a.tf\n M b.tf\nMM c.tf\nA  d.tf\nD  e.tf\n"))

	// Empty and whitespace
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("\n\n\n"))

	// Paths with spaces (git quotes these)
	f.Add([]byte(" M \"path with spaces.tf\"\n"))
	f.Add([]byte("?? path with spaces.tf\n"))

	// Unicode paths
	f.Add([]byte(" M 日本語.tf\n"))
	f.Add([]byte("?? 中文文件.tf\n"))
	f.Add([]byte("A  émoji🎉.tf\n"))

	// Malformed lines (too short)
	f.Add([]byte("X\n"))
	f.Add([]byte("XX\n"))
	f.Add([]byte("X \n"))
	f.Add([]byte("  \n"))
	f.Add([]byte("   \n"))

	// Unknown status codes (should still parse)
	f.Add([]byte("ZZ unknown.tf\n"))
	f.Add([]byte("99 numeric.tf\n"))
	f.Add([]byte("@@ special.tf\n"))

	// Binary garbage
	f.Add([]byte{0x00, 0x01, 0x02})
	f.Add([]byte("XX\x00file.tf\n"))

	// Mixed line endings
	f.Add([]byte(" M file1.tf\r\n M file2.tf\r\n"))
	f.Add([]byte("?? file.tf\r"))

	// Very long paths
	f.Add([]byte("M  " + strings.Repeat("a/", 50) + "file.tf\n"))

	// Edge case: line exactly 3 chars (minimum valid: "XY path")
	f.Add([]byte("M  \n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Exercise the same parsing logic as GetFileStatuses (git.go:341-365)
		// without the git command execution. Uses real FileStatus type.
		var statuses []FileStatus
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) >= 3 {
				status := strings.TrimSpace(line[:2])
				path := strings.TrimSpace(line[3:])
				if path != "" {
					statuses = append(statuses, FileStatus{
						Path:   path,
						Status: status,
					})
				}
			}
		}
		_ = scanner.Err()

		// Dereference results to catch nil/empty field panics
		for _, s := range statuses {
			_ = len(s.Path)
			_ = len(s.Status)
		}
	})
}

// FuzzValidateGitRef tests the git ref validation regex.
// It exercises the pattern matching with arbitrary ref names.
func FuzzValidateGitRef(f *testing.F) {
	// Valid refs
	f.Add("main")
	f.Add("master")
	f.Add("origin/main")
	f.Add("feature/add-tests")
	f.Add("v1.0.0")
	f.Add("HEAD")
	f.Add("HEAD@{1}")
	f.Add("HEAD@{upstream}")
	f.Add("refs/heads/main")
	f.Add("abc123")
	f.Add("release-1.0")
	f.Add("feature_branch")
	f.Add("")

	// Invalid refs (shell injection attempts)
	f.Add("; rm -rf /")
	f.Add("$(whoami)")
	f.Add("`id`")
	f.Add("main; echo pwned")
	f.Add("main | cat /etc/passwd")
	f.Add("main && echo pwned")
	f.Add("main\necho pwned")
	f.Add("main'")
	f.Add("main\"")
	f.Add("$(cat /etc/passwd)")
	f.Add("main > /tmp/out")
	f.Add("main < /etc/passwd")

	// Unicode (should be invalid)
	f.Add("分支")
	f.Add("ветка")
	f.Add("émoji🎉")

	// Control characters
	f.Add("ref\x00name")
	f.Add("ref\nname")
	f.Add("ref\tname")

	// Very long refs
	f.Add(strings.Repeat("a", 1000))
	f.Add("feature/" + strings.Repeat("x", 500))

	f.Fuzz(func(t *testing.T, ref string) {
		// ValidateGitRef should never panic regardless of input
		err := ValidateGitRef(ref)
		// If valid, the regex matched; if invalid, error returned
		// Either way, no panic is expected
		_ = err
	})
}

// FuzzSanitizeGitError tests the error message sanitization.
// It exercises path removal and truncation with arbitrary error messages.
func FuzzSanitizeGitError(f *testing.F) {
	// Nil error (edge case)
	// Note: We test with string seeds and wrap in errors.New

	// Simple errors
	f.Add("git command failed")
	f.Add("fatal: not a git repository")
	f.Add("error: pathspec 'nonexistent' did not match any file(s)")

	// Errors with paths
	f.Add("fatal: could not read '/home/user/secret/repo/.git/config'")
	f.Add("error: could not read '/etc/passwd' or '/home/user/.ssh/id_rsa'")
	f.Add("fatal: cannot lock ref 'refs/heads/main': '/home/user/repo/.git/refs/heads/main.lock' exists")

	// Multiple paths
	f.Add("error: '/path/a' and '/path/b' differ")
	f.Add("diff '/home/a/file' '/home/b/file'")

	// Very long errors
	f.Add(strings.Repeat("x", 600))
	f.Add("error: " + strings.Repeat("/path/segment/", 50) + "file")

	// Unicode
	f.Add("fatal: 找不到文件 '/home/用户/文件'")
	f.Add("ошибка: '/путь/файл' не найден")

	// Special characters
	f.Add("error: path contains 'quotes' and \"double quotes\"")
	f.Add("error: newline\nin message")
	f.Add("error: tab\tin message")

	// Binary-ish
	f.Add("error: \x00\x01\x02")
	f.Add("git: \xff\xfe")

	f.Fuzz(func(t *testing.T, msg string) {
		// sanitizeGitError should never panic regardless of input
		err := errors.New(msg)
		result := sanitizeGitError(err)
		// Result should be a valid error (non-nil for non-nil input)
		if result == nil {
			t.Errorf("sanitizeGitError returned nil for non-nil input")
		}
		// Result should be truncated if too long.
		// The suffix "... (truncated)" is 15 chars, so max length is maxGitErrorLen + 15.
		// Note: path replacement may change length before truncation, but the final
		// result should still respect the truncation limit.
		const truncationSuffix = "... (truncated)" // 15 chars
		maxLen := maxGitErrorLen + len(truncationSuffix)
		if len(result.Error()) > maxLen {
			t.Errorf("sanitizeGitError did not truncate long message: len=%d, max=%d", len(result.Error()), maxLen)
		}

		// Dereference result to exercise string operations
		_ = len(result.Error())
	})
}

// FuzzFilterTerraformFiles tests the Terraform file filtering logic.
// It exercises extension checking with arbitrary file paths.
func FuzzFilterTerraformFiles(f *testing.F) {
	// Terraform files
	f.Add("main.tf")
	f.Add("variables.tf")
	f.Add("terraform.tfvars")
	f.Add(".tflint.hcl")
	f.Add("config.hcl")

	// Non-Terraform files
	f.Add("README.md")
	f.Add("config.yaml")
	f.Add("main.go")
	f.Add("test.py")
	f.Add("file.txt")

	// Edge cases
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add(".tf")
	f.Add(".hcl")
	f.Add(".tfvars")
	f.Add("notf")
	f.Add("tffile")
	f.Add("hclfile")

	// Case variations
	f.Add("Main.TF")
	f.Add("File.Tf")
	f.Add("VAR.TFVARS")
	f.Add("CONFIG.HCL")

	// Paths with extensions in directory names
	f.Add(".tf/file.txt")
	f.Add("module.tf/main.go")
	f.Add("terraform.hcl/config.yaml")

	// Unicode
	f.Add("日本語.tf")
	f.Add("中文.hcl")
	f.Add("файл.tfvars")

	// Special characters
	f.Add("file with spaces.tf")
	f.Add("file-with-dashes.tf")
	f.Add("file_with_underscores.tf")
	f.Add("file@special.tf")

	// Very long paths
	f.Add(strings.Repeat("a/", 50) + "file.tf")

	// Multiple extensions
	f.Add("file.tf.bak")
	f.Add("file.hcl.old")
	f.Add("backup.tfvars.1")

	f.Fuzz(func(t *testing.T, path string) {
		// Exercise filterTerraformFiles logic with single file
		g := &Git{}
		files := []string{path}
		result := g.filterTerraformFiles(files)
		// Result should be 0 or 1 files, no panic expected
		if len(result) > 1 {
			t.Errorf("filterTerraformFiles returned more files than input")
		}

		// Dereference results to catch nil/empty element panics
		for _, f := range result {
			_ = len(f)
		}
	})
}
