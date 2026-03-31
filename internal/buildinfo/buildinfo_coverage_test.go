package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionConsistency(t *testing.T) {
	// GetVersion() must return the same value as the Version package var
	assert.Equal(t, Version, GetVersion())
}

func TestInitSetsAllVariables(t *testing.T) {
	// After init(), all variables should be non-empty (set by embedded file, ldflags, or debug info)
	assert.NotEmpty(t, Version, "Version should be set after init")
	assert.NotEmpty(t, Commit, "Commit should be set after init")
	assert.NotEmpty(t, Date, "Date should be set after init")
}

func TestVersionFileEmbedded(t *testing.T) {
	// The version.json embed should exist (may be empty in dev builds)
	// We can't test the actual init logic since it runs at package load,
	// but we can verify the embed variable exists
	_ = versionFile // just verify it compiles
}
