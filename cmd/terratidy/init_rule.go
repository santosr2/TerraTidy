package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	initRuleName   string
	initRuleType   string
	initRuleOutput string
)

var initRuleCmd = &cobra.Command{
	Use:   "init-rule",
	Short: "Initialize a new custom rule",
	Long:  `Generate scaffolding for a new custom rule in Go, YAML, or Bash.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🎨 Creating %s rule: %s...\n", initRuleType, initRuleName)
		// TODO: Implement init-rule logic
		return nil
	},
}

func init() {
	initRuleCmd.Flags().StringVar(&initRuleName, "name", "", "rule name (required)")
	initRuleCmd.Flags().StringVar(&initRuleType, "type", "go", "rule type (go|yaml|bash|tflint-plugin)")
	initRuleCmd.Flags().StringVar(&initRuleOutput, "output", "", "output directory")
	initRuleCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(initRuleCmd)
}

