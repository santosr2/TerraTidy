package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// If version is still "dev", try to get version from build info
	// This works when installed via `go install` from a tagged version
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Get module version (e.g., v0.1.0)
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}

			// Get VCS info from build settings
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) >= 7 {
						commit = setting.Value[:7]
					} else if setting.Value != "" {
						commit = setting.Value
					}
				case "vcs.time":
					if setting.Value != "" {
						date = setting.Value
					}
				case "vcs.modified":
					if setting.Value == "true" && commit != "none" {
						commit += "-dirty"
					}
				}
			}
		}
	}
}

func main() {
	if err := Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
