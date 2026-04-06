package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func BenchmarkGitIsGitRepo(b *testing.B) {
	// Use current directory (should be in a git repo)
	git := NewGit(".")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = git.IsGitRepo()
	}
}

func BenchmarkGitGetChangedFiles(b *testing.B) {
	// Create a temp git repo
	tmpDir := b.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		b.Skip("git not available")
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(testFile, []byte("# initial"), 0o644); err != nil {
		b.Fatal(err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Modify the file to create a change
	if err := os.WriteFile(testFile, []byte("# modified"), 0o644); err != nil {
		b.Fatal(err)
	}

	git := NewGit(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = git.GetChangedFiles("HEAD")
	}
}

func BenchmarkGitGetStagedFiles(b *testing.B) {
	tmpDir := b.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		b.Skip("git not available")
	}

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "main.tf")
	if err := os.WriteFile(testFile, []byte("# initial"), 0o644); err != nil {
		b.Fatal(err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create new file and stage it
	newFile := filepath.Join(tmpDir, "variables.tf")
	if err := os.WriteFile(newFile, []byte("# variables"), 0o644); err != nil {
		b.Fatal(err)
	}

	cmd = exec.Command("git", "add", newFile)
	cmd.Dir = tmpDir
	_ = cmd.Run()

	git := NewGit(tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = git.GetStagedFiles()
	}
}
