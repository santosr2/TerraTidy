//go:build !windows

package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_LoadFromDirectory_PermissionIssues(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	restrictedDir := filepath.Join(tmpDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0o000)
	require.NoError(t, err)

	// Restore permissions after test
	t.Cleanup(func() {
		_ = os.Chmod(restrictedDir, 0o755)
	})

	manager := NewManager([]string{restrictedDir})
	err = manager.LoadAll()
	// Should return permission error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
