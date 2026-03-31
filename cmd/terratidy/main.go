package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/santosr2/terratidy/internal/buildinfo"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// Version variables - set from the buildinfo package which uses
// embedded version.json (from goreleaser) or falls back to ldflags/debug info.
var (
	version = buildinfo.Version
	commit  = buildinfo.Commit
	date    = buildinfo.Date
)

func main() {
	if err := Execute(); err != nil {
		var exitErr *sdk.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
