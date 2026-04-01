package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionConsistency(t *testing.T) {
	assert.Equal(t, Version, GetVersion())
}

func TestInitSetsAllVariables(t *testing.T) {
	assert.NotEmpty(t, Version, "Version should be set after init")
	assert.NotEmpty(t, Commit, "Commit should be set after init")
	assert.NotEmpty(t, Date, "Date should be set after init")
}

func TestResolveVersion_EmbeddedFile(t *testing.T) {
	// Save and restore globals
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "dev", "none", "unknown"
	embedded := []byte(`{"version":"1.2.3","commit":"abc1234","date":"2026-01-01T00:00:00Z"}`)
	resolveVersion(embedded)

	assert.Equal(t, "1.2.3", Version)
	assert.Equal(t, "abc1234", Commit)
	assert.Equal(t, "2026-01-01T00:00:00Z", Date)
}

func TestResolveVersion_EmbeddedPartialFields(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "dev", "none", "unknown"
	embedded := []byte(`{"version":"2.0.0"}`)
	resolveVersion(embedded)

	assert.Equal(t, "2.0.0", Version)
	assert.Equal(t, "none", Commit, "should keep default when field is empty")
	assert.Equal(t, "unknown", Date, "should keep default when field is empty")
}

func TestResolveVersion_InvalidJSON(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "dev", "none", "unknown"
	embedded := []byte(`not json`)
	resolveVersion(embedded)

	// Invalid JSON falls through to debug info; version may change from "dev"
	// but should not panic
	assert.NotEmpty(t, Version)
}

func TestResolveVersion_EmptyFile(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "dev", "none", "unknown"
	resolveVersion(nil)

	// Nil file falls through to debug info
	assert.NotEmpty(t, Version)
}

func TestResolveVersion_DebugInfoFallback(t *testing.T) {
	// When Version is "dev" and no embedded file, resolveVersion uses debug.ReadBuildInfo
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "dev", "none", "unknown"
	resolveVersion(nil)

	// In test binary, debug.ReadBuildInfo returns (devel) for version,
	// but may set Commit/Date from VCS settings
	// The key assertion is that it doesn't panic and Version is set
	assert.NotEmpty(t, Version)
}

func TestResolveVersion_LdflagsAlreadySet(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version, Commit, Date = "1.0.0-ldflags", "def5678", "2025-06-01"
	resolveVersion(nil)

	// ldflags version is not "dev", so it should be preserved
	assert.Equal(t, "1.0.0-ldflags", Version)
	assert.Equal(t, "def5678", Commit)
	assert.Equal(t, "2025-06-01", Date)
}
