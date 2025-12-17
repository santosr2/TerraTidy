package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	styleFix   bool
	styleCheck bool
	styleDiff  bool
)

var styleCmd = &cobra.Command{
	Use:   "style",
	Short: "Check and fix style issues",
	Long:  `Run the Style Engine to check for style violations and optionally fix them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Checking style...")
		// TODO: Implement style logic
		return nil
	},
}

func init() {
	styleCmd.Flags().BoolVar(&styleFix, "fix", false, "automatically fix style issues")
	styleCmd.Flags().BoolVar(&styleCheck, "check", false, "check only, do not modify files")
	styleCmd.Flags().BoolVar(&styleDiff, "diff", false, "show diff of style changes")
	rootCmd.AddCommand(styleCmd)
}

