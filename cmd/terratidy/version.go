package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	versionShort bool
	versionJSON  bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, build, and runtime information for TerraTidy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionJSON {
			fmt.Printf(`{
  "version": "%s",
  "commit": "%s",
  "date": "%s",
  "goVersion": "%s",
  "platform": "%s/%s"
}
`, version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
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
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "output in JSON format")
	rootCmd.AddCommand(versionCmd)
}
