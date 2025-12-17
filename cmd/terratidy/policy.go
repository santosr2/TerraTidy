package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Run policy checks",
	Long:  `Run OPA policy checks against Terraform configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🛡️  Running policy checks...")
		// TODO: Implement policy logic
		return nil
	},
}

func init() {
	rootCmd.AddCommand(policyCmd)
}

