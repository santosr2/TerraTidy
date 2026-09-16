package main

import (
	"os"

	"github.com/santosr2/TerraTidy/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	cfgFile           string
	profile           string
	format            string
	changed           bool
	noRecurse         bool
	absolutePaths     bool
	severityThreshold string
	color             bool
	excludePatterns   []string
)

var rootCmd = &cobra.Command{
	Use:   "terratidy",
	Short: "TerraTidy - Terraform/Terragrunt Quality Platform",
	Long: `TerraTidy is a comprehensive quality platform for Terraform and Terragrunt.

It provides formatting, style checking, linting, and policy enforcement
in a single binary with no external dependencies.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// Resolve the final --color value once per invocation, honoring an
	// explicit flag, NO_COLOR, and FORCE_COLOR before falling back to TTY
	// auto-detection. Every subcommand reads the already-resolved `color`
	// var and passes it on as an explicit formatter parameter.
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		color = output.ResolveColor(
			color,
			cmd.Flags().Changed("color"),
			os.Getenv("NO_COLOR"),
			os.Getenv("FORCE_COLOR"),
			term.IsTerminal(int(os.Stdout.Fd())),
		)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .terratidy.yaml)")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "profile to use from config")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "output format (text|table|json|json-compact|sarif|html|junit|markdown|github)")
	rootCmd.PersistentFlags().BoolVar(&changed, "changed", false, "only check changed files")
	rootCmd.PersistentFlags().BoolVar(&noRecurse, "no-recurse", false, "disable recursive directory traversal")
	rootCmd.PersistentFlags().BoolVar(&absolutePaths, "absolute-paths", false, "output absolute file paths instead of relative")
	rootCmd.PersistentFlags().StringVar(
		&severityThreshold, "severity-threshold", "",
		"minimum severity level to fail (info|warning|error)",
	)
	rootCmd.PersistentFlags().BoolVar(
		&color, "color",
		// The default is the auto-detected value rather than a hardcoded true,
		// so callers that never reach PersistentPreRunE still get color that
		// matches the destination. PersistentPreRunE re-resolves on every run to
		// apply an explicit flag; for an unchanged flag it returns this same value.
		output.ResolveColor(
			true, false,
			os.Getenv("NO_COLOR"),
			os.Getenv("FORCE_COLOR"),
			term.IsTerminal(int(os.Stdout.Fd())),
		),
		"enable colored output (default: auto-detected from the terminal, NO_COLOR, and FORCE_COLOR)",
	)
	rootCmd.PersistentFlags().StringSliceVar(
		&excludePatterns, "exclude", nil,
		"glob patterns to exclude (repeatable or comma-separated)",
	)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
