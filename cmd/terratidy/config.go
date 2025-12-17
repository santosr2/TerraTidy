package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage TerraTidy configuration files.`,
}

var configSplitCmd = &cobra.Command{
	Use:   "split",
	Short: "Split configuration into modular structure",
	Long:  `Convert a single .terratidy.yaml file into a modular directory structure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📂 Splitting configuration...")
		// TODO: Implement config split logic
		return nil
	},
}

var configMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge split configurations into single file",
	Long:  `Combine modular configuration files into a single .terratidy.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔀 Merging configurations...")
		// TODO: Implement config merge logic
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration",
	Long:  `Display the final configuration after all imports and merges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📄 Showing resolved configuration...")
		// TODO: Implement config show logic
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  `Validate the configuration file and all imports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✓ Validating configuration...")
		// TODO: Implement config validate logic
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSplitCmd)
	configCmd.AddCommand(configMergeCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}

