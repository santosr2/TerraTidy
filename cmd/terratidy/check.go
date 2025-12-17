package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run all checks (fmt, style, lint, policy)",
	Long:  `Run all enabled engines in check mode. This is the recommended command for CI/CD.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✅ Running all checks...")
		// TODO: Implement check logic
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

