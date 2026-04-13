//go:build !windows

package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Symlink security tests verify that validateWorkspacePath correctly handles symlinks.
// These tests require actual filesystem operations and are skipped on Windows
// where symlinks require elevated privileges.

func TestValidateWorkspacePath_SymlinkEscape(t *testing.T) {
	// Security test: a symlink inside the workspace that points to a file
	// outside the workspace should be blocked.

	// Create the "outside" directory first
	outsideDir := t.TempDir()
	secretFile := filepath.Join(outsideDir, "secret.tf")
	require.NoError(t, os.WriteFile(secretFile, []byte("# secret data"), 0o644))

	// Create the workspace directory
	workspaceDir := t.TempDir()

	// Create a symlink inside workspace pointing to file outside workspace
	symlinkPath := filepath.Join(workspaceDir, "escape.tf")
	require.NoError(t, os.Symlink(secretFile, symlinkPath))

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = workspaceDir

	// The symlink path looks innocent, but resolves to outside the workspace
	_, err := server.validateWorkspacePath(symlinkPath)

	require.Error(t, err, "symlink pointing outside workspace should be blocked")
	assert.Contains(t, err.Error(), "escapes workspace")
}

func TestValidateWorkspacePath_SymlinkWithinWorkspace(t *testing.T) {
	// Positive test: a symlink inside the workspace that points to another
	// location inside the workspace should be allowed.

	workspaceDir := t.TempDir()

	// Create a real file in a subdirectory
	modulesDir := filepath.Join(workspaceDir, "modules")
	require.NoError(t, os.MkdirAll(modulesDir, 0o755))
	realFile := filepath.Join(modulesDir, "main.tf")
	require.NoError(t, os.WriteFile(realFile, []byte("# module code"), 0o644))

	// Create a symlink in the workspace root pointing to the file in modules/
	symlinkPath := filepath.Join(workspaceDir, "linked.tf")
	require.NoError(t, os.Symlink(realFile, symlinkPath))

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = workspaceDir

	result, err := server.validateWorkspacePath(symlinkPath)

	require.NoError(t, err, "symlink within workspace should be allowed")
	// The result should be the resolved path (the real file, not the symlink)
	// Use EvalSymlinks on expected to handle macOS /var -> /private/var symlink
	expectedResolved, err := filepath.EvalSymlinks(realFile)
	require.NoError(t, err)
	assert.Equal(t, expectedResolved, result, "should return resolved path")
}

func TestValidateWorkspacePath_SymlinkRelativeEscape(t *testing.T) {
	// Security test: a symlink using relative path that escapes workspace.

	// Create parent directory to hold both workspace and target
	parentDir := t.TempDir()

	// Create target outside workspace
	outsideFile := filepath.Join(parentDir, "outside", "secret.tf")
	require.NoError(t, os.MkdirAll(filepath.Dir(outsideFile), 0o755))
	require.NoError(t, os.WriteFile(outsideFile, []byte("# secret"), 0o644))

	// Create workspace as a subdirectory
	workspaceDir := filepath.Join(parentDir, "workspace")
	require.NoError(t, os.MkdirAll(workspaceDir, 0o755))

	// Create a symlink using relative path: ../outside/secret.tf
	symlinkPath := filepath.Join(workspaceDir, "relative-escape.tf")
	require.NoError(t, os.Symlink("../outside/secret.tf", symlinkPath))

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = workspaceDir

	_, err := server.validateWorkspacePath(symlinkPath)

	require.Error(t, err, "relative symlink escaping workspace should be blocked")
	assert.Contains(t, err.Error(), "escapes workspace")
}

func TestValidateWorkspacePath_SymlinkChain(t *testing.T) {
	// Security test: a chain of symlinks where the final target is outside workspace.

	outsideDir := t.TempDir()
	secretFile := filepath.Join(outsideDir, "secret.tf")
	require.NoError(t, os.WriteFile(secretFile, []byte("# secret"), 0o644))

	workspaceDir := t.TempDir()

	// Create symlink chain: link1.tf -> link2.tf -> secret.tf (outside)
	link2 := filepath.Join(workspaceDir, "link2.tf")
	require.NoError(t, os.Symlink(secretFile, link2))

	link1 := filepath.Join(workspaceDir, "link1.tf")
	require.NoError(t, os.Symlink(link2, link1))

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = workspaceDir

	// EvalSymlinks resolves the entire chain
	_, err := server.validateWorkspacePath(link1)

	require.Error(t, err, "symlink chain escaping workspace should be blocked")
	assert.Contains(t, err.Error(), "escapes workspace")
}

func TestValidateWorkspacePath_SymlinkedWorkspaceRoot(t *testing.T) {
	// Edge case: workspace root itself is a symlink to a real directory.
	// Files within should still be accessible.

	realWorkspaceDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(realWorkspaceDir, "main.tf"),
		[]byte("# main"),
		0o644,
	))

	// Create a symlink to the workspace directory
	parentDir := t.TempDir()
	symlinkWorkspace := filepath.Join(parentDir, "workspace-link")
	require.NoError(t, os.Symlink(realWorkspaceDir, symlinkWorkspace))

	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	server.workspaceRoot = symlinkWorkspace

	// Access file via the symlinked workspace path
	filePath := filepath.Join(symlinkWorkspace, "main.tf")
	result, err := server.validateWorkspacePath(filePath)

	require.NoError(t, err, "file in symlinked workspace should be accessible")
	// Result is the resolved path (through real workspace directory)
	// Use EvalSymlinks on expected to handle macOS /var -> /private/var symlink
	expectedResolved, err := filepath.EvalSymlinks(filepath.Join(realWorkspaceDir, "main.tf"))
	require.NoError(t, err)
	assert.Equal(t, expectedResolved, result, "should return resolved path")
}
