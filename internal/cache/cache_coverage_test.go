package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentAccess(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	dir := t.TempDir()

	// Create test files
	var files []string
	for i := range 5 {
		f := writeTempTF(t, dir, "test"+string(rune('0'+i))+".tf", `variable "v`+string(rune('0'+i))+`" {}`)
		files = append(files, f)
	}

	// Concurrent reads and writes
	var wg sync.WaitGroup
	for _, f := range files {
		wg.Add(2)
		go func(path string) {
			defer wg.Done()
			_, _ = c.GetOrParse(path)
		}(f)
		go func(path string) {
			defer wg.Done()
			c.Get(path)
		}(f)
	}
	wg.Wait()

	assert.LessOrEqual(t, c.Size(), 5)
}

func TestConcurrentGetOrParse(t *testing.T) {
	c := New(Options{MaxAge: time.Hour, MaxSize: 100})
	dir := t.TempDir()
	f := writeTempTF(t, dir, "test.tf", `resource "test" "t" {}`)

	// Multiple goroutines parsing the same file
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.GetOrParse(f)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}
