package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock is a controllable clock for deterministic tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// writeTempTF creates a .tf file in the given dir and returns its path.
func writeTempTF(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestCacheGetSet(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `resource "aws_instance" "test" {}`)

	entry, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(entry.Content), "aws_instance")
	assert.NotNil(t, entry.File)

	cached, ok := c.Get(tmpFile)
	require.True(t, ok)
	assert.Equal(t, entry.Hash, cached.Hash)
}

func TestCacheExpiry(t *testing.T) {
	clk := newFakeClock(time.Now())
	c := New(Options{MaxAge: 10 * time.Second, MaxSize: 100, Clock: clk})
	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `variable "test" {}`)

	_, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)

	_, ok := c.Get(tmpFile)
	require.True(t, ok, "entry should be cached")

	// Advance past expiry
	clk.Advance(11 * time.Second)

	_, ok = c.Get(tmpFile)
	assert.False(t, ok, "entry should be expired")
}

func TestCacheFileModification(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	dir := t.TempDir()
	tmpFile := writeTempTF(t, dir, "test.tf", `output "test" {}`)

	_, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)

	// Overwrite with different content and a newer mod time.
	// Use os.Chtimes to guarantee the mod time differs from the cached one,
	// regardless of filesystem timestamp resolution.
	newContent := []byte(`output "modified" {}`)
	require.NoError(t, os.WriteFile(tmpFile, newContent, 0o644))
	future := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(tmpFile, future, future))

	_, ok := c.Get(tmpFile)
	assert.False(t, ok, "cache should be invalidated after file modification")
}

func TestCacheMaxSize(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 2})
	dir := t.TempDir()

	for i := range 3 {
		f := writeTempTF(t, dir, "test"+string(rune('0'+i))+".tf", `variable "v`+string(rune('0'+i))+`" {}`)
		_, err := c.GetOrParse(f)
		require.NoError(t, err)
	}

	assert.Equal(t, 2, c.Size(), "oldest entry should be evicted")
}

func TestCacheDisabled(t *testing.T) {
	c := New(Options{Disabled: true})
	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `locals { test = true }`)

	_, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)

	_, ok := c.Get(tmpFile)
	assert.False(t, ok, "entry should not be cached when disabled")
	assert.Equal(t, 0, c.Size())
}

func TestCacheClear(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `data "test" "d" {}`)

	_, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, 1, c.Size())

	c.Clear()
	assert.Equal(t, 0, c.Size())
}

func TestCacheStats(t *testing.T) {
	c := New(Options{MaxAge: 5 * time.Minute, MaxSize: 50})
	stats := c.Stats()

	assert.Equal(t, 50, stats.MaxSize)
	assert.Equal(t, 5*time.Minute, stats.MaxAge)
	assert.False(t, stats.Disabled)
	assert.Equal(t, 0, stats.Entries)
}

func TestCacheDelete(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `module "test" {}`)

	_, err := c.GetOrParse(tmpFile)
	require.NoError(t, err)

	c.Delete(tmpFile)

	_, ok := c.Get(tmpFile)
	assert.False(t, ok, "entry should be deleted")
}

func TestDefaultCache(t *testing.T) {
	ResetDefault()
	c := Default()
	require.NotNil(t, c)
	assert.False(t, c.Stats().Disabled)
}

func TestHashContent(t *testing.T) {
	hash1 := hashContent([]byte("hello"))
	hash2 := hashContent([]byte("hello"))
	hash3 := hashContent([]byte("world"))

	assert.Equal(t, hash1, hash2, "same content should produce same hash")
	assert.NotEqual(t, hash1, hash3, "different content should produce different hash")
	assert.Len(t, hash1, 64, "SHA256 hex should be 64 chars")
}

func TestConfigureDefault(t *testing.T) {
	// Reset to ensure clean state
	ResetDefault()

	// Verify default settings
	stats := Default().Stats()
	assert.Equal(t, 5*time.Minute, stats.MaxAge)
	assert.Equal(t, 1000, stats.MaxSize)
	assert.False(t, stats.Disabled)

	// Configure with custom options
	ConfigureDefault(Options{
		MaxAge:   10 * time.Minute,
		MaxSize:  500,
		Disabled: false,
	})

	stats = Default().Stats()
	assert.Equal(t, 10*time.Minute, stats.MaxAge)
	assert.Equal(t, 500, stats.MaxSize)
	assert.False(t, stats.Disabled)

	// Configure as disabled
	ConfigureDefault(Options{Disabled: true})
	assert.True(t, Default().Stats().Disabled)

	// Reset back to defaults for other tests
	ResetDefault()
}

func TestConfigureDefault_MaxAgeAffectsTTL(t *testing.T) {
	clk := newFakeClock(time.Now())

	// Configure with 10 minute TTL
	ConfigureDefault(Options{
		MaxAge:  10 * time.Minute,
		MaxSize: 100,
		Clock:   clk,
	})
	defer ResetDefault()

	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `resource "test" "a" {}`)

	// Cache the file
	_, err := Default().GetOrParse(tmpFile)
	require.NoError(t, err)

	// Should still be cached at 9 minutes
	clk.Advance(9 * time.Minute)
	_, ok := Default().Get(tmpFile)
	assert.True(t, ok, "entry should still be cached before TTL")

	// Should be expired at 11 minutes
	clk.Advance(2 * time.Minute)
	_, ok = Default().Get(tmpFile)
	assert.False(t, ok, "entry should be expired after TTL")
}

func TestConfigureDefault_MaxSizeLimitsEntries(t *testing.T) {
	// Configure with max 3 entries
	ConfigureDefault(Options{
		MaxAge:  time.Hour,
		MaxSize: 3,
	})
	defer ResetDefault()

	dir := t.TempDir()
	files := make([]string, 5)
	for i := range files {
		files[i] = writeTempTF(t, dir, fmt.Sprintf("test%d.tf", i), fmt.Sprintf(`resource "r" "r%d" {}`, i))
	}

	// Cache all 5 files
	for _, f := range files {
		_, err := Default().GetOrParse(f)
		require.NoError(t, err)
	}

	// Cache should have at most 3 entries
	assert.LessOrEqual(t, Default().Size(), 3, "cache should not exceed max size")
}

func TestConfigureDefault_DisabledBypassesCache(t *testing.T) {
	ConfigureDefault(Options{Disabled: true})
	defer ResetDefault()

	tmpFile := writeTempTF(t, t.TempDir(), "test.tf", `variable "x" {}`)

	// GetOrParse should work but not cache
	entry, err := Default().GetOrParse(tmpFile)
	require.NoError(t, err)
	assert.NotNil(t, entry)

	// Get should return false (not cached)
	_, ok := Default().Get(tmpFile)
	assert.False(t, ok, "disabled cache should not store entries")

	// Size should be 0
	assert.Equal(t, 0, Default().Size(), "disabled cache should have no entries")
}
