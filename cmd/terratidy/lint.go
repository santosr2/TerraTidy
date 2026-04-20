package main

import (
	"context"
	"fmt"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/internal/engines/lint"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	lintConfigFile string
	lintPlugins    []string
	lintRules      []string
)

var lintCmd = &cobra.Command{
	Use:   "lint [paths...]",
	Short: "Run linting checks",
	Long: `Run linting checks to detect errors and best practice violations.

Linting is a read-only analysis that never modifies files. Unlike fmt and style,
there is no --check or --diff flag because lint only reports issues without
making changes. To fix lint issues, you must edit your Terraform code manually.

Lint performs static analysis to find potential issues, security vulnerabilities,
and violations of best practices.

Use --changed to only lint files that have been modified in git.`,
	Example: `  # Lint current directory
  terratidy lint

  # Lint specific paths
  terratidy lint ./modules ./environments

  # Only lint changed files
  terratidy lint --changed

  # Enable specific rules
  terratidy lint --rule terraform_required_version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			return err // Already wrapped as ExitConfig by loadConfig
		}

		// Get target files (respecting --changed flag and excludes)
		files, err := getTargetFilesWithExcludes(args, changed, cfg.Exclude, cfg)
		if err != nil {
			return sdk.NewInternalError(fmt.Errorf("finding files: %w", err))
		}

		if len(files) == 0 {
			printNoFilesMessage()
			return nil
		}

		// Load plugin rules
		pluginRules, err := loadPluginRules(cfg)
		if err != nil {
			return sdk.NewConfigError(fmt.Errorf("loading plugins: %w", err))
		}

		// Build lint config from terratidy config, then apply CLI overrides
		lintCfg := buildLintConfig(cfg)

		// CLI flags override config file settings (use Changed() to detect explicit flags)
		if cmd.Flags().Changed("tflint-config") {
			lintCfg.ConfigFile = lintConfigFile
		}
		if cmd.Flags().Changed("plugin") {
			lintCfg.Plugins = lintPlugins
		}

		// Create rule config from CLI flags
		if len(lintRules) > 0 {
			lintCfg.Rules = make(map[string]lint.RuleConfig)
			for _, rule := range lintRules {
				lintCfg.Rules[rule] = lint.RuleConfig{
					Enabled:  config.BoolPtr(true),
					Severity: "warning",
				}
			}
		}

		// Create lint engine with plugin rules
		engine := lint.New(lintCfg, pluginRules...)

		// For structured output formats, skip the progress messages
		useStructuredOutput := format != "" && format != "text"

		if !useStructuredOutput {
			modeMsg := ""
			if changed {
				modeMsg = " (changed files only)"
			}
			fmt.Printf("Running linter on %s%s...\n\n", formatFileCount(len(files)), modeMsg)
		}

		findings, err := engine.Run(context.Background(), files)
		if err != nil {
			return sdk.NewInternalError(fmt.Errorf("running linter: %w", err))
		}

		// Apply severity threshold filtering
		threshold := getEffectiveSeverityThreshold(cfg)
		findings = filterFindingsBySeverity(findings, threshold)

		// Output results using formatter
		return outputLintResults(findings, cfg)
	},
}

func outputLintResults(findings []sdk.Finding, cfg *config.Config) error {
	return outputResults(findings, "Lint summary", cfg)
}

func init() {
	lintCmd.Flags().StringVar(&lintConfigFile, "tflint-config", ".tflint.hcl", "path to TFLint config file")
	lintCmd.Flags().StringSliceVar(&lintPlugins, "plugin", []string{}, "plugins to enable (aws, google, azurerm)")
	lintCmd.Flags().StringSliceVar(&lintRules, "rule", []string{}, "specific rules to enable")
	rootCmd.AddCommand(lintCmd)
}
