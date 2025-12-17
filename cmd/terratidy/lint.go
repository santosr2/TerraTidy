package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run linting checks",
	Long:  `Run TFLint to check for errors and best practice violations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔎 Running linter...")
		// TODO: Implement lint logic
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

