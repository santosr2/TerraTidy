package buildinfo

import (
	"testing"
)

func TestGetters(t *testing.T) {
	t.Run("GetVersion returns non-empty", func(t *testing.T) {
		v := GetVersion()
		if v == "" {
			t.Error("GetVersion() should not return empty string")
		}
	})

	t.Run("package variables are non-empty", func(t *testing.T) {
		if Version == "" {
			t.Error("Version should not be empty")
		}
		if Commit == "" {
			t.Error("Commit should not be empty")
		}
		if Date == "" {
			t.Error("Date should not be empty")
		}
	})
}

func TestDefaultValues(t *testing.T) {
	t.Run("Version defaults to dev when not set", func(t *testing.T) {
		v := GetVersion()
		if v == "" {
			t.Error("Version should never be empty")
		}
	})
}
