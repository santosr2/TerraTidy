package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentEviction verifies that eviction during concurrent access
// doesn't cause race conditions or data corruption.
// Run with -race to detect race conditions.
func TestConcurrentEviction(t *testing.T) {
	// Small cache to force frequent eviction
	const maxSize = 5
	c := New(Options{MaxAge: time.Hour, MaxSize: maxSize})
	dir := t.TempDir()

	// Create more files than cache capacity
	const numFiles = 20
	var files []string
	for i := range numFiles {
		f := writeTempTF(t, dir, fmt.Sprintf("evict%d.tf", i), fmt.Sprintf(`variable "v%d" {}`, i))
		files = append(files, f)
	}

	var wg sync.WaitGroup

	// Concurrent reads and writes that trigger eviction
	for i, f := range files {
		// Launch reader
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			_, _ = c.GetOrParse(path)
		}(f)

		// Launch writer that triggers eviction
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			entry := &Entry{
				Content:  []byte(fmt.Sprintf(`variable "set%d" {}`, idx)),
				CachedAt: time.Now(),
			}
			c.Set(path+".set", entry)
		}(i, f)
	}

	wg.Wait()

	// Cache should never exceed maxSize
	size := c.Size()
	assert.LessOrEqual(t, size, maxSize, "cache size should not exceed maxSize")
	assert.GreaterOrEqual(t, size, 0, "cache size should be non-negative")
}

// TestConcurrentEvictionUnderPressure simulates high contention with
// many goroutines trying to add/read/evict simultaneously.
func TestConcurrentEvictionUnderPressure(t *testing.T) {
	const maxSize = 3 // Very small cache
	c := New(Options{MaxAge: time.Hour, MaxSize: maxSize})
	dir := t.TempDir()

	// Create test files
	const numFiles = 10
	var files []string
	for i := range numFiles {
		f := writeTempTF(t, dir, fmt.Sprintf("pressure%d.tf", i), fmt.Sprintf(`output "o%d" {}`, i))
		files = append(files, f)
	}

	var wg sync.WaitGroup
	const numOps = 100

	// Many concurrent operations
	for i := range numOps {
		wg.Add(1)
		go func(op int) {
			defer wg.Done()

			// Mix of operations
			switch op % 4 {
			case 0: // Get
				c.Get(files[op%numFiles])
			case 1: // GetOrParse
				_, _ = c.GetOrParse(files[op%numFiles])
			case 2: // Set with new path (triggers eviction)
				entry := &Entry{
					Content:  []byte(fmt.Sprintf(`# op %d`, op)),
					CachedAt: time.Now(),
				}
				c.Set(fmt.Sprintf("%s/virtual%d.tf", dir, op), entry)
			case 3: // Delete
				c.Delete(files[op%numFiles])
			}
		}(i)
	}

	wg.Wait()

	// Cache should be in consistent state
	size := c.Size()
	assert.LessOrEqual(t, size, maxSize, "cache should not exceed maxSize after pressure test")
	assert.GreaterOrEqual(t, size, 0, "cache size should be non-negative")
}

// TestConcurrentEvictionWithExpiry tests eviction combined with time-based expiry
func TestConcurrentEvictionWithExpiry(t *testing.T) {
	// Use existing fakeClock from cache_test.go
	clock := newFakeClock(time.Now())

	const maxSize = 5
	c := New(Options{
		MaxAge:  100 * time.Millisecond,
		MaxSize: maxSize,
		Clock:   clock,
	})
	dir := t.TempDir()

	// Create test files
	const numFiles = 10
	var files []string
	for i := range numFiles {
		f := writeTempTF(t, dir, fmt.Sprintf("expiry%d.tf", i), fmt.Sprintf(`locals { x = %d }`, i))
		files = append(files, f)
	}

	var wg sync.WaitGroup

	// Fill cache
	for _, f := range files[:maxSize] {
		_, err := c.GetOrParse(f)
		require.NoError(t, err)
	}

	// Advance time past expiry
	clock.Advance(200 * time.Millisecond)

	// Concurrent access after expiry
	for _, f := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			// GetOrParse should re-parse expired entries
			_, _ = c.GetOrParse(path)
		}(f)
	}

	wg.Wait()

	// Cache should still be bounded
	size := c.Size()
	assert.LessOrEqual(t, size, maxSize, "cache should be bounded after expiry test")
}
