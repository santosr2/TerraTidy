package main

import (
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/buildinfo"
)

// Version variables - these are set from the buildinfo package
// which uses embedded version.json (from goreleaser) or falls back to ldflags/debug info
var (
	version = buildinfo.Version
	commit  = buildinfo.Commit
	date    = buildinfo.Date
)

func main() {
	// Re-read from buildinfo in case init order matters
	version = buildinfo.Version
	commit = buildinfo.Commit
	date = buildinfo.Date

	if err := Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
