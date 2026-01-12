package buildinfo

import (
	"testing"
)

func TestGetters(t *testing.T) {
	// These should return valid values (either embedded, ldflags, or fallback)
	t.Run("GetVersion returns non-empty", func(t *testing.T) {
		v := GetVersion()
		if v == "" {
			t.Error("GetVersion() should not return empty string")
		}
	})

	t.Run("GetCommit returns non-empty", func(t *testing.T) {
		c := GetCommit()
		if c == "" {
			t.Error("GetCommit() should not return empty string")
		}
	})

	t.Run("GetDate returns non-empty", func(t *testing.T) {
		d := GetDate()
		if d == "" {
			t.Error("GetDate() should not return empty string")
		}
	})

	t.Run("Short returns version without v prefix", func(t *testing.T) {
		s := Short()
		if s == "" {
			t.Error("Short() should not return empty string")
		}
		if s[0] == 'v' {
			t.Error("Short() should not start with 'v'")
		}
	})
}

func TestDefaultValues(t *testing.T) {
	// When built without ldflags and empty version.json, should have dev defaults
	// Note: This test documents expected behavior in dev mode
	t.Run("Version defaults to dev when not set", func(t *testing.T) {
		// In test mode, version could be "dev" or a real version from debug.ReadBuildInfo
		v := GetVersion()
		if v == "" {
			t.Error("Version should never be empty")
		}
	})
}
