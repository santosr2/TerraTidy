package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Rule management commands",
	Long:  `Manage and inspect TerraTidy rules.`,
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available rules",
	Long:  `Display all built-in and custom rules.`,
	RunE: func(cmd *cobra.Command, args []string) error{
		fmt.Println("📋 Listing available rules...")
		// TODO: Implement rules list logic
		return nil
	},
}

var rulesDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate rule documentation",
	Long:  `Generate markdown documentation for all rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📚 Generating rule documentation...")
		// TODO: Implement rules docs logic
		return nil
	},
}

func init() {
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesDocsCmd)
	rootCmd.AddCommand(rulesCmd)
}

