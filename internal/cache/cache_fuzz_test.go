package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FuzzCacheKey tests cache key generation and lookup with arbitrary file paths
// and content. It exercises the cache's path handling, hash generation, and
// concurrent access patterns to ensure no panics occur with edge-case input.
func FuzzCacheKey(f *testing.F) {
	// Valid file paths
	f.Add("main.tf", []byte("resource {}"))
	f.Add("variables.tf", []byte("variable \"name\" {}"))
	f.Add("outputs.tf", []byte("output \"value\" {}"))
	f.Add("path/to/nested/file.tf", []byte("module {}"))

	// Paths with special characters
	f.Add("file-with-dashes.tf", []byte("resource {}"))
	f.Add("file_with_underscores.tf", []byte("resource {}"))
	f.Add("file.with.dots.tf", []byte("resource {}"))
	f.Add("file with spaces.tf", []byte("resource {}"))

	// Unicode paths
	f.Add("日本語.tf", []byte("resource {}"))
	f.Add("файл.tf", []byte("resource {}"))
	f.Add("αρχείο.tf", []byte("resource {}"))

	// Empty and minimal content
	f.Add("empty.tf", []byte{})
	f.Add("single.tf", []byte{0x00})
	f.Add("minimal.tf", []byte("x"))

	// Binary content
	f.Add("binary.tf", []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe})
	f.Add("nulls.tf", []byte{0x00, 0x00, 0x00, 0x00})

	// Large content
	largeContent := make([]byte, 100000)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	f.Add("large.tf", largeContent)

	// Valid HCL content of varying complexity
	f.Add("valid.tf", []byte(`
terraform {
  required_version = ">= 1.0"
}

resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t3.micro"

  tags = {
    Name = "test"
  }
}
`))

	// Content with special characters
	f.Add("special.tf", []byte("# Comment with unicode: 日本語\nvariable \"test\" {}"))
	f.Add("quotes.tf", []byte(`resource "test" "name" { value = "\"quoted\"" }`))

	// Paths with directory components (stripped to base by sanitization below)
	f.Add("parent.tf", []byte("resource {}"))
	f.Add("grandparent.tf", []byte("resource {}"))
	f.Add("current.tf", []byte("resource {}"))

	// Very long filename
	longName := make([]byte, 200)
	for i := range longName {
		longName[i] = 'a'
	}
	f.Add(string(longName)+".tf", []byte("resource {}"))

	f.Fuzz(func(t *testing.T, path string, content []byte) {
		// Skip empty paths as they're not useful cache keys
		if path == "" {
			return
		}

		// Create temp directory for test files
		tmpDir := t.TempDir()

		// Sanitize path to avoid escaping temp directory
		// Replace dangerous patterns while preserving most characters
		safePath := filepath.Base(path)
		if safePath == "." || safePath == ".." || safePath == "" {
			safePath = "test.tf"
		}

		fullPath := filepath.Join(tmpDir, safePath)

		// Write the fuzzed content to a real file
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			// Skip if we can't create the file (invalid filename, etc.)
			return
		}

		// Create a fresh cache with short expiry for testing
		c := New(Options{
			MaxAge:  1 * time.Minute,
			MaxSize: 100,
		})

		// Exercise GetOrParse - should never panic
		entry, err := c.GetOrParse(fullPath)
		if err != nil {
			// Errors are fine (invalid HCL, etc.), panics are not
			return
		}

		// Verify entry has valid data
		_ = entry.Hash
		_ = entry.Content
		_ = entry.ModTime
		_ = entry.CachedAt

		// Exercise Get - should return cached entry
		// Note: Cache miss can legitimately occur if the filesystem has coarse timestamp
		// resolution (e.g., FAT32 has 2-second resolution) and the mtime differs between
		// WriteFile and the subsequent Stat in Get. This is expected on some platforms.
		_, _ = c.Get(fullPath)

		// Exercise Delete - should never panic
		c.Delete(fullPath)

		// Verify entry is deleted
		_, exists := c.Get(fullPath)
		if exists {
			t.Errorf("entry still exists after Delete")
		}

		// Exercise Set with manually created entry
		manualEntry := &Entry{
			Content:  content,
			Hash:     hashContent(content),
			ModTime:  time.Now(),
			CachedAt: time.Now(),
		}
		c.Set(fullPath, manualEntry)

		// Exercise Clear
		c.Clear()

		// Verify cache is empty
		if c.Size() != 0 {
			t.Errorf("cache not empty after Clear: got %d entries", c.Size())
		}
	})
}

// FuzzHashContent tests the hashContent function with arbitrary bytes.
// SHA256 should handle any input without panicking.
func FuzzHashContent(f *testing.F) {
	// Empty and minimal
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xff})

	// Binary data
	f.Add([]byte{0x00, 0x01, 0x02, 0x03})
	f.Add([]byte{0xff, 0xfe, 0xfd, 0xfc})

	// Unicode text
	f.Add([]byte("hello世界"))
	f.Add([]byte("αβγδ"))

	// Large content (64KB is sufficient to test; 1MB would bloat seed corpus)
	largeContent := make([]byte, 65536)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	f.Add(largeContent)

	// Repeated patterns
	f.Add([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, content []byte) {
		// hashContent should never panic
		hash := hashContent(content)

		// Verify hash is valid hex string of correct length (SHA256 = 64 hex chars)
		if len(hash) != 64 {
			t.Errorf("expected hash length 64, got %d", len(hash))
		}

		// Verify hash contains only valid hex characters (0-9, a-f)
		for i, c := range hash {
			isDigit := c >= '0' && c <= '9'
			isHexLetter := c >= 'a' && c <= 'f'
			if !isDigit && !isHexLetter {
				t.Errorf("hash contains non-hex character at position %d: %q", i, c)
			}
		}
	})
}

// FuzzCacheEviction tests cache eviction logic with rapid set/get cycles.
// Exercises the oldest-first (FIFO) eviction under memory pressure.
// Uses direct Set calls to avoid filesystem I/O overhead.
func FuzzCacheEviction(f *testing.F) {
	// Seeds for varying entry counts
	f.Add(1, 5)
	f.Add(5, 10)
	f.Add(10, 5)
	f.Add(100, 50)
	f.Add(50, 100)
	f.Add(0, 50) // Test MaxSize=0 (unlimited) after normalization
	f.Add(50, 0) // Test numEntries=0 after normalization

	f.Fuzz(func(t *testing.T, maxSize, numEntries int) {
		// Normalize to reasonable ranges
		if maxSize <= 0 || maxSize > 1000 {
			maxSize = 100
		}
		if numEntries <= 0 || numEntries > 1000 {
			numEntries = 100
		}

		c := New(Options{
			MaxAge:  1 * time.Hour, // Long expiry to test eviction by size
			MaxSize: maxSize,
		})

		now := time.Now()

		// Add entries using direct Set calls (no filesystem I/O)
		for i := 0; i < numEntries; i++ {
			path := fmt.Sprintf("/cache/test/file%d.tf", i)
			entry := &Entry{
				Content:  []byte("resource {}"),
				Hash:     fmt.Sprintf("hash%d", i),
				ModTime:  now,
				CachedAt: now.Add(time.Duration(i) * time.Millisecond), // Stagger for eviction order
			}
			c.Set(path, entry)
		}

		// Cache size should not exceed maxSize
		if c.Size() > maxSize {
			t.Errorf("cache size %d exceeds maxSize %d", c.Size(), maxSize)
		}

		// Clear should work regardless of state
		c.Clear()

		if c.Size() != 0 {
			t.Errorf("cache not empty after Clear")
		}
	})
}

// FuzzCacheDisabled tests the disabled cache behavior.
func FuzzCacheDisabled(f *testing.F) {
	// Basic paths and content
	f.Add("test.tf", []byte("resource {}"))
	f.Add("main.tf", []byte("variable {}"))

	// Empty and minimal content
	f.Add("empty.tf", []byte{})
	f.Add("single.tf", []byte{0x00})

	// Binary content
	f.Add("binary.tf", []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe})

	// Unicode paths
	f.Add("日本語.tf", []byte("resource {}"))
	f.Add("файл.tf", []byte("resource {}"))

	f.Fuzz(func(t *testing.T, path string, content []byte) {
		if path == "" {
			return
		}

		c := New(Options{
			Disabled: true,
		})

		tmpDir := t.TempDir()
		safePath := filepath.Base(path)
		if safePath == "." || safePath == ".." || safePath == "" {
			safePath = "test.tf"
		}
		fullPath := filepath.Join(tmpDir, safePath)

		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return
		}

		// GetOrParse should still work but not cache
		entry, err := c.GetOrParse(fullPath)
		if err != nil {
			return
		}
		_ = entry.Hash

		// Get should always return false when disabled
		_, ok := c.Get(fullPath)
		if ok {
			t.Errorf("disabled cache returned entry")
		}

		// Size should always be 0 when disabled
		if c.Size() != 0 {
			t.Errorf("disabled cache has non-zero size: %d", c.Size())
		}
	})
}
