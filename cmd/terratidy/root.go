package main

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile           string
	profile           string
	format            string
	changed           bool
	paths             []string
	severityThreshold string
)

var rootCmd = &cobra.Command{
	Use:   "terratidy",
	Short: "TerraTidy - Terraform/Terragrunt Quality Platform",
	Long: `TerraTidy is a comprehensive quality platform for Terraform and Terragrunt.

It provides formatting, style checking, linting, and policy enforcement
in a single binary with no external dependencies.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .terratidy.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "profile to use from config")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "output format (text|json|sarif)")
	rootCmd.PersistentFlags().BoolVar(&changed, "changed", false, "only check changed files")
	rootCmd.PersistentFlags().StringSliceVar(&paths, "paths", []string{}, "paths to check")
	rootCmd.PersistentFlags().StringVar(&severityThreshold, "severity-threshold", "", "minimum severity level to fail (info|warning|error)")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
