package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	initInteractive bool
	initSplit       bool
	initMonorepo    bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TerraTidy configuration",
	Long:  `Create a .terratidy.yaml configuration file with recommended settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🚀 Initializing TerraTidy configuration...")
		// TODO: Implement init logic
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "interactive configuration setup")
	initCmd.Flags().BoolVar(&initSplit, "split", false, "create modular split configuration")
	initCmd.Flags().BoolVar(&initMonorepo, "monorepo", false, "set up for monorepo")
	rootCmd.AddCommand(initCmd)
}

