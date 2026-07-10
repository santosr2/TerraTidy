//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Symlink tests are skipped on Windows because symlinks require elevated privileges.
// See: https://docs.microsoft.com/en-us/windows/security/threat-protection/security-policy-settings/create-symbolic-links

func TestFindHCLFiles_SymlinkedFilesIncluded(t *testing.T) {
	// Note: filepath.Walk visits symlink entries (but does not follow symlinked directories).
	// This test validates that symlinked .tf files are included in the results, which works
	// because filepath.Walk passes symlink entries to the callback and sdk.IsHCLFile checks
	// the path string (not the symlink target type).

	tmpDir := t.TempDir()

	// Create a real .tf file in a subdirectory
	realDir := filepath.Join(tmpDir, "real")
	require.NoError(t, os.MkdirAll(realDir, 0o755))
	realFile := filepath.Join(realDir, "main.tf")
	require.NoError(t, os.WriteFile(realFile, []byte("# real file"), 0o644))

	// Create a symlink to the .tf file in the main directory
	symlinkFile := filepath.Join(tmpDir, "linked.tf")
	require.NoError(t, os.Symlink(realFile, symlinkFile))

	t.Run("recursive", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir}, true)
		require.NoError(t, err)
		assert.Len(t, files, 2, "should find both real and symlinked .tf files")

		foundSymlink := false
		for _, f := range files {
			if filepath.Base(f) == "linked.tf" {
				foundSymlink = true
				break
			}
		}
		assert.True(t, foundSymlink, "symlinked file should be found")
	})

	t.Run("non-recursive", func(t *testing.T) {
		// Non-recursive scans use os.ReadDir which also includes symlink entries
		files, err := findHCLFiles([]string{tmpDir}, false)
		require.NoError(t, err)
		assert.Len(t, files, 1, "should find symlinked .tf file in top-level dir")

		foundSymlink := false
		for _, f := range files {
			if filepath.Base(f) == "linked.tf" {
				foundSymlink = true
				break
			}
		}
		assert.True(t, foundSymlink, "symlinked file should be found in non-recursive mode")
	})
}

func TestFindHCLFiles_SymlinkDirectory(t *testing.T) {
	// Note: filepath.Walk does NOT follow symlinked directories. This test pins that behavior.
	// If we want to change this behavior in the future, we would need to use filepath.WalkDir
	// with FollowSymlinks or implement custom symlink resolution.

	tmpDir := t.TempDir()

	// Create a real directory with a .tf file
	realDir := filepath.Join(tmpDir, "real-modules")
	require.NoError(t, os.MkdirAll(realDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "main.tf"), []byte("# real"), 0o644))

	// Create a symlink to the directory
	symlinkDir := filepath.Join(tmpDir, "linked-modules")
	require.NoError(t, os.Symlink(realDir, symlinkDir))

	// Create a top-level .tf file so we know the scan is working
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.tf"), []byte("# root"), 0o644))

	t.Run("recursive does not follow symlinked directories", func(t *testing.T) {
		files, err := findHCLFiles([]string{tmpDir}, true)
		require.NoError(t, err)

		// filepath.Walk does not follow symlinked directories, so we should find:
		// - root.tf (top-level)
		// - real-modules/main.tf (real directory)
		// But NOT linked-modules/main.tf (symlinked directory is not traversed)
		assert.Len(t, files, 2, "should find files in real dir but not traverse symlinked dir")

		for _, f := range files {
			assert.NotContains(t, f, "linked-modules", "symlinked directory should not be traversed")
		}
	})

	t.Run("non-recursive ignores symlinked directories", func(t *testing.T) {
		// Non-recursive only scans top-level files, so neither directory is traversed
		files, err := findHCLFiles([]string{tmpDir}, false)
		require.NoError(t, err)
		assert.Len(t, files, 1, "should only find root.tf")
		assert.Equal(t, filepath.Join(tmpDir, "root.tf"), files[0])
		assert.NotContains(t, files[0], "linked-modules", "symlinked directory should not appear")
	})
}

func TestFindHCLFiles_BrokenSymlink(t *testing.T) {
	// Note: Both filepath.Walk (recursive) and os.ReadDir (non-recursive) use os.Lstat
	// internally, which succeeds for broken symlinks (reads symlink metadata, not target).
	// Broken symlinks with .tf extension are included in results. This test pins that
	// behavior. If we want to skip broken symlinks, we would need to add explicit
	// checking with os.Stat or filepath.EvalSymlinks.

	tmpDir := t.TempDir()

	// Create a real .tf file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "valid.tf"), []byte("# valid"), 0o644))

	// Create a broken symlink (points to non-existent file)
	brokenSymlink := filepath.Join(tmpDir, "broken.tf")
	require.NoError(t, os.Symlink("/non/existent/target.tf", brokenSymlink))

	t.Run("recursive includes broken symlinks", func(t *testing.T) {
		// filepath.Walk uses Lstat, so broken symlinks are visited as entries
		files, err := findHCLFiles([]string{tmpDir}, true)
		require.NoError(t, err)

		// Both valid.tf and broken.tf should be found (Lstat succeeds for both)
		assert.Len(t, files, 2, "broken symlinks with .tf extension are included")

		foundBroken := false
		for _, f := range files {
			if filepath.Base(f) == "broken.tf" {
				foundBroken = true
				break
			}
		}
		assert.True(t, foundBroken, "broken symlink should be found")
	})

	t.Run("non-recursive includes broken symlinks", func(t *testing.T) {
		// os.ReadDir also uses Lstat internally, same behavior as filepath.Walk
		files, err := findHCLFiles([]string{tmpDir}, false)
		require.NoError(t, err)
		assert.Len(t, files, 2, "broken symlinks with .tf extension are included")

		foundBroken := false
		for _, f := range files {
			if filepath.Base(f) == "broken.tf" {
				foundBroken = true
				break
			}
		}
		assert.True(t, foundBroken, "broken symlink should be found in non-recursive mode")
	})
}
