package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/santosr2/TerraTidy/internal/buildinfo"
	"github.com/santosr2/TerraTidy/pkg/sdk"
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
			// NewFindingsError sets Err=nil (output is the formatter's job); only
			// config/internal exits carry a non-nil Err that the user needs to see.
			if exitErr.Err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", exitErr.Err)
			}
			os.Exit(exitErr.Code)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
