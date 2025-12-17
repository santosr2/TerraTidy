package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	testRuleFixtures string
	testRuleExpect   string
)

var testRuleCmd = &cobra.Command{
	Use:   "test-rule [rule-name]",
	Short: "Test a specific rule",
	Long:  `Test a rule against fixtures and expected findings.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleName := args[0]
		fmt.Printf("🧪 Testing rule: %s...\n", ruleName)
		// TODO: Implement test-rule logic
		return nil
	},
}

func init() {
	testRuleCmd.Flags().StringVar(&testRuleFixtures, "fixtures", "test_fixtures/", "fixtures directory")
	testRuleCmd.Flags().StringVar(&testRuleExpect, "expect", "", "expected findings file")
	rootCmd.AddCommand(testRuleCmd)
}

