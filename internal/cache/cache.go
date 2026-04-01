// Package cache provides file content and parsed HCL caching for TerraTidy.
// It enables sharing parsed HCL files across multiple engine executions,
// reducing redundant file reads and parsing operations.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// Entry represents a cached file entry
type Entry struct {
	Content   []byte
	File      *hcl.File
	ModTime   time.Time
	Hash      string
	CachedAt  time.Time
	ParseErrs hcl.Diagnostics
}

// Clock abstracts time for testing. Defaults to the real clock.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// FileCache provides thread-safe caching of file contents and parsed HCL
type FileCache struct {
	mu       sync.RWMutex
	entries  map[string]*Entry
	maxAge   time.Duration
	maxSize  int
	disabled bool
	clock    Clock
}

// Options configures the cache behavior
type Options struct {
	MaxAge   time.Duration // Maximum age of cache entries (0 = no expiry)
	MaxSize  int           // Maximum number of entries (0 = unlimited)
	Disabled bool          // Disable caching entirely
	Clock    Clock         // Time source (defaults to real clock, override for tests)
}

// DefaultOptions returns sensible default cache options
func DefaultOptions() Options {
	return Options{
		MaxAge:   5 * time.Minute,
		MaxSize:  1000,
		Disabled: false,
	}
}

// New creates a new FileCache with the given options
func New(opts Options) *FileCache {
	clk := opts.Clock
	if clk == nil {
		clk = realClock{}
	}
	return &FileCache{
		entries:  make(map[string]*Entry),
		maxAge:   opts.MaxAge,
		maxSize:  opts.MaxSize,
		disabled: opts.Disabled,
		clock:    clk,
	}
}

// NewDefault creates a new FileCache with default options
func NewDefault() *FileCache {
	return New(DefaultOptions())
}

// Get retrieves a cached entry if it exists and is still valid
func (c *FileCache) Get(path string) (*Entry, bool) {
	if c.disabled {
		return nil, false
	}

	c.mu.RLock()
	entry, exists := c.entries[path]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if c.maxAge > 0 && c.clock.Now().Sub(entry.CachedAt) > c.maxAge {
		c.Delete(path)
		return nil, false
	}

	// Verify file hasn't changed
	info, err := os.Stat(path)
	if err != nil {
		c.Delete(path)
		return nil, false
	}

	if !info.ModTime().Equal(entry.ModTime) {
		c.Delete(path)
		return nil, false
	}

	return entry, true
}

// GetOrParse retrieves a cached entry or parses the file if not cached
func (c *FileCache) GetOrParse(path string) (*Entry, error) {
	// Try to get from cache first
	if entry, ok := c.Get(path); ok {
		return entry, nil
	}

	// Read and parse the file
	entry, err := c.parseFile(path)
	if err != nil {
		return nil, err
	}

	// Store in cache if not disabled
	if !c.disabled {
		c.Set(path, entry)
	}

	return entry, nil
}

// Set stores an entry in the cache
func (c *FileCache) Set(path string, entry *Entry) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if c.maxSize > 0 && len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[path] = entry
}

// Delete removes an entry from the cache
func (c *FileCache) Delete(path string) {
	c.mu.Lock()
	delete(c.entries, path)
	c.mu.Unlock()
}

// Clear removes all entries from the cache
func (c *FileCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*Entry)
	c.mu.Unlock()
}

// Size returns the current number of cached entries
func (c *FileCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics
func (c *FileCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		Entries:  len(c.entries),
		MaxSize:  c.maxSize,
		MaxAge:   c.maxAge,
		Disabled: c.disabled,
	}
}

// Stats holds cache statistics.
type Stats struct {
	Entries  int
	MaxSize  int
	MaxAge   time.Duration
	Disabled bool
}

// parseFile reads and parses a file
func (c *FileCache) parseFile(path string) (*Entry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, path)

	entry := &Entry{
		Content:   content,
		File:      file,
		ModTime:   info.ModTime(),
		Hash:      hashContent(content),
		CachedAt:  c.clock.Now(),
		ParseErrs: diags,
	}

	return entry, nil
}

// evictOldest removes the oldest entry from the cache
// Must be called with lock held
func (c *FileCache) evictOldest() {
	var oldestPath string
	var oldestTime time.Time

	for path, entry := range c.entries {
		if oldestPath == "" || entry.CachedAt.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.CachedAt
		}
	}

	if oldestPath != "" {
		delete(c.entries, oldestPath)
	}
}

// hashContent returns a SHA256 hash of the content
func hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// Global default cache instance
var defaultCache = NewDefault()

// Default returns the global default cache instance
func Default() *FileCache {
	return defaultCache
}

// ResetDefault resets the global default cache (useful for testing)
func ResetDefault() {
	defaultCache = NewDefault()
}
