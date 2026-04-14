package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var versionShort bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, build, and runtime information for TerraTidy.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		isJSON := format == "json" || format == "json-compact"

		// Detect conflicting flags
		if versionShort && isJSON {
			return fmt.Errorf("--short cannot be combined with --format %s", format)
		}

		// Reject unsupported formats (version only supports text and JSON variants)
		if format != "text" && !isJSON {
			return fmt.Errorf("unsupported format %q for version command (supported: text, json, json-compact)", format)
		}

		// Use global format flag (json or json-compact)
		if isJSON {
			versionInfo := map[string]string{
				"version":    version,
				"commit":     commit,
				"date":       date,
				"go_version": runtime.Version(),
				"platform":   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}

			var data []byte
			var err error
			if format == "json-compact" {
				data, err = json.Marshal(versionInfo)
			} else {
				data, err = json.MarshalIndent(versionInfo, "", "  ")
			}
			if err != nil {
				return fmt.Errorf("marshaling version info: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if versionShort {
			fmt.Println(version)
			return nil
		}

		// Default detailed output
		fmt.Printf("TerraTidy version %s\n", version)
		fmt.Printf("  Commit:      %s\n", commit)
		fmt.Printf("  Build date:  %s\n", date)
		fmt.Printf("  Go version:  %s\n", runtime.Version())
		fmt.Printf("  Platform:    %s/%s\n", runtime.GOOS, runtime.GOARCH)

		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "print only version number")
	rootCmd.AddCommand(versionCmd)
}
