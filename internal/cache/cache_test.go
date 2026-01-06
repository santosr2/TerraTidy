package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	cache := New(Options{MaxAge: time.Hour, MaxSize: 100})

	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`resource "aws_instance" "test" {}`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Parse and cache
	entry, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	if string(entry.Content) != string(content) {
		t.Errorf("content mismatch: got %s, want %s", entry.Content, content)
	}

	if entry.File == nil {
		t.Error("expected parsed HCL file, got nil")
	}

	// Get from cache
	cachedEntry, ok := cache.Get(tmpFile)
	if !ok {
		t.Fatal("expected entry to be cached")
	}

	if cachedEntry.Hash != entry.Hash {
		t.Error("cached entry hash mismatch")
	}
}

func TestCacheExpiry(t *testing.T) {
	cache := New(Options{MaxAge: 10 * time.Millisecond, MaxSize: 100})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`variable "test" {}`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Parse and cache
	_, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Should be cached
	if _, ok := cache.Get(tmpFile); !ok {
		t.Error("expected entry to be cached")
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get(tmpFile); ok {
		t.Error("expected entry to be expired")
	}
}

func TestCacheFileModification(t *testing.T) {
	cache := New(Options{MaxAge: time.Hour, MaxSize: 100})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`output "test" {}`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Parse and cache
	_, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Modify the file (need to wait a bit for modtime to change)
	time.Sleep(10 * time.Millisecond)
	newContent := []byte(`output "modified" {}`)
	if err := os.WriteFile(tmpFile, newContent, 0o644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Should detect modification and invalidate
	if _, ok := cache.Get(tmpFile); ok {
		t.Error("expected cache to be invalidated after file modification")
	}
}

func TestCacheMaxSize(t *testing.T) {
	cache := New(Options{MaxAge: time.Hour, MaxSize: 2})

	tmpDir := t.TempDir()

	// Create 3 files
	for i := 0; i < 3; i++ {
		tmpFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".tf")
		content := []byte(`variable "v` + string(rune('0'+i)) + `" {}`)
		if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		if _, err := cache.GetOrParse(tmpFile); err != nil {
			t.Fatalf("failed to parse file: %v", err)
		}
	}

	// Should only have 2 entries (oldest evicted)
	if cache.Size() != 2 {
		t.Errorf("expected 2 entries, got %d", cache.Size())
	}
}

func TestCacheDisabled(t *testing.T) {
	cache := New(Options{Disabled: true})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`locals { test = true }`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Parse (should not cache)
	_, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	// Should not be cached
	if _, ok := cache.Get(tmpFile); ok {
		t.Error("expected entry to not be cached when disabled")
	}

	if cache.Size() != 0 {
		t.Errorf("expected 0 entries when disabled, got %d", cache.Size())
	}
}

func TestCacheClear(t *testing.T) {
	cache := New(Options{MaxAge: time.Hour, MaxSize: 100})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`data "test" "d" {}`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	if cache.Size() != 1 {
		t.Errorf("expected 1 entry, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", cache.Size())
	}
}

func TestCacheStats(t *testing.T) {
	opts := Options{MaxAge: 5 * time.Minute, MaxSize: 50, Disabled: false}
	cache := New(opts)

	stats := cache.Stats()

	if stats.MaxSize != 50 {
		t.Errorf("expected MaxSize 50, got %d", stats.MaxSize)
	}
	if stats.MaxAge != 5*time.Minute {
		t.Errorf("expected MaxAge 5m, got %v", stats.MaxAge)
	}
	if stats.Disabled {
		t.Error("expected Disabled to be false")
	}
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Entries)
	}
}

func TestCacheDelete(t *testing.T) {
	cache := New(Options{MaxAge: time.Hour, MaxSize: 100})

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tf")
	content := []byte(`module "test" {}`)
	if err := os.WriteFile(tmpFile, content, 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := cache.GetOrParse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	cache.Delete(tmpFile)

	if _, ok := cache.Get(tmpFile); ok {
		t.Error("expected entry to be deleted")
	}
}

func TestDefaultCache(t *testing.T) {
	ResetDefault()
	cache := Default()

	if cache == nil {
		t.Fatal("expected default cache to be non-nil")
	}

	stats := cache.Stats()
	if stats.Disabled {
		t.Error("expected default cache to be enabled")
	}
}

func TestHashContent(t *testing.T) {
	content1 := []byte("hello")
	content2 := []byte("hello")
	content3 := []byte("world")

	hash1 := hashContent(content1)
	hash2 := hashContent(content2)
	hash3 := hashContent(content3)

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different content should produce different hash")
	}
	if len(hash1) != 64 { // SHA256 hex = 64 chars
		t.Errorf("expected hash length 64, got %d", len(hash1))
	}
}
