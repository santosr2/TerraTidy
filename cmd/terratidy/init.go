// Package main provides the init command for TerraTidy.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/spf13/cobra"
)

var (
	initInteractive bool
	initSplit       bool
	initMonorepo    bool
	initForce       bool
)

// customConfigOptions holds the options for generating a custom configuration.
type customConfigOptions struct {
	fmtEnabled    bool
	styleEnabled  bool
	lintEnabled   bool
	policyEnabled bool
	severity      string
	failFast      bool
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TerraTidy configuration",
	Long: `Create a .terratidy.yaml configuration file with recommended settings.

This command creates a configuration file in the current directory with
sensible defaults. Use --interactive for a guided setup experience.`,
	Example: `  # Initialize with defaults
  terratidy init

  # Interactive setup
  terratidy init --interactive

  # Create split (modular) configuration
  terratidy init --split

  # Set up for monorepo
  terratidy init --monorepo`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "interactive configuration setup")
	initCmd.Flags().BoolVar(&initSplit, "split", false, "create modular split configuration")
	initCmd.Flags().BoolVar(&initMonorepo, "monorepo", false, "set up for monorepo")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration")
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	configPath := ".terratidy.yaml"

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return sdk.NewConfigError(fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath))
	}

	fmt.Println("Initializing TerraTidy configuration...")
	fmt.Println()

	var config string

	if initInteractive {
		var err error
		config, err = interactiveInit()
		if err != nil {
			// I/O failure (broken stdin, etc.), not user-correctable config issue.
			return sdk.NewInternalError(fmt.Errorf("interactive setup failed: %w", err))
		}
	} else if initSplit {
		return initSplitConfig()
	} else if initMonorepo {
		config = generateMonorepoConfig()
	} else {
		config = generateDefaultConfig()
	}

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return sdk.NewInternalError(fmt.Errorf("writing config file: %w", err))
	}

	fmt.Printf("Created %s\n\n", configPath)
	fmt.Println("Next steps:")
	fmt.Println("  1. Review and customize the configuration")
	fmt.Println("  2. Run 'terratidy check' to verify your Terraform code")
	fmt.Println("  3. Run 'terratidy fix' to automatically fix issues")
	fmt.Println()

	return nil
}

// interactiveInit runs an interactive configuration setup.
func interactiveInit() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to TerraTidy interactive setup!")
	fmt.Println()

	// Ask about engines
	fmt.Print("Enable formatting checks? [Y/n]: ")
	fmtEnabled, err := readYesNo(reader, true)
	if err != nil {
		return "", fmt.Errorf("reading format preference: %w", err)
	}

	fmt.Print("Enable style checks? [Y/n]: ")
	styleEnabled, err := readYesNo(reader, true)
	if err != nil {
		return "", fmt.Errorf("reading style preference: %w", err)
	}

	fmt.Print("Enable linting? [Y/n]: ")
	lintEnabled, err := readYesNo(reader, true)
	if err != nil {
		return "", fmt.Errorf("reading lint preference: %w", err)
	}

	fmt.Print("Enable policy checks? [y/N]: ")
	policyEnabled, err := readYesNo(reader, false)
	if err != nil {
		return "", fmt.Errorf("reading policy preference: %w", err)
	}

	// Ask about severity
	fmt.Print("Minimum severity threshold (info/warning/error) [warning]: ")
	severity, err := readLine(reader)
	if err != nil {
		return "", fmt.Errorf("reading severity: %w", err)
	}
	if severity == "" {
		severity = "warning"
	}

	// Ask about fail-fast
	fmt.Print("Stop on first error? [y/N]: ")
	failFast, err := readYesNo(reader, false)
	if err != nil {
		return "", fmt.Errorf("reading fail-fast preference: %w", err)
	}

	fmt.Println()

	opts := customConfigOptions{
		fmtEnabled:    fmtEnabled,
		styleEnabled:  styleEnabled,
		lintEnabled:   lintEnabled,
		policyEnabled: policyEnabled,
		severity:      severity,
		failFast:      failFast,
	}
	return generateCustomConfig(opts), nil
}

func readYesNo(reader *bufio.Reader, defaultYes bool) (bool, error) {
	line, err := readLine(reader)
	if err != nil {
		return false, err
	}
	if line == "" {
		return defaultYes, nil
	}
	return strings.EqualFold(line, "y") || strings.EqualFold(line, "yes"), nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	// ReadString returns data AND error when EOF is hit without delimiter.
	// Accept partial data if present (e.g., "echo -n y | terratidy init -i").
	if err != nil && (err != io.EOF || len(line) == 0) {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// initSplitConfig creates a modular split configuration.
func initSplitConfig() error {
	// Create .terratidy directory
	configDir := ".terratidy"
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return sdk.NewInternalError(fmt.Errorf("creating config directory: %w", err))
	}

	// Create main config file
	mainConfig := `# TerraTidy Configuration
# This file imports modular configurations from the .terratidy directory.
version: 1

imports:
  - ".terratidy/*.yaml"

# Global settings
severity_threshold: warning
fail_fast: false
parallel: true
`

	// Create engine configs
	fmtConfig := `# Formatting Configuration
engines:
  fmt:
    enabled: true
`

	styleConfig := `# Style Configuration
engines:
  style:
    enabled: true
    # Customize style rules here
    # rules:
    #   style.blank-line-between-blocks:
    #     enabled: true
    #     severity: warning
`

	lintConfig := `# Linting Configuration
engines:
  lint:
    enabled: true
    # Enable specific plugins
    # plugins:
    #   - aws
    #   - google
`

	policyConfig := `# Policy Configuration
engines:
  policy:
    enabled: false  # Enable when you have policies defined
    # Policy directories
    # policy_dirs:
    #   - ./policies
`

	// Write all files
	files := map[string]string{
		".terratidy.yaml":                       mainConfig,
		filepath.Join(configDir, "fmt.yaml"):    fmtConfig,
		filepath.Join(configDir, "style.yaml"):  styleConfig,
		filepath.Join(configDir, "lint.yaml"):   lintConfig,
		filepath.Join(configDir, "policy.yaml"): policyConfig,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return sdk.NewInternalError(fmt.Errorf("writing %s: %w", path, err))
		}
		fmt.Printf("Created %s\n", path)
	}

	fmt.Println()
	fmt.Println("Split configuration created!")
	fmt.Println("Edit individual files in .terratidy/ to customize each engine.")
	fmt.Println()

	return nil
}

// generateDefaultConfig generates the default configuration.
func generateDefaultConfig() string {
	return `# TerraTidy Configuration
# Documentation: https://github.com/santosr2/TerraTidy
version: 1

# Global settings
severity_threshold: warning
fail_fast: false
parallel: true

# Engine configurations
engines:
  fmt:
    enabled: true

  style:
    enabled: true

  lint:
    enabled: true

  policy:
    enabled: false
    # policy_dirs:
    #   - ./policies

# Per-engine rule configuration
# engines:
#   style:
#     rules:
#       style.blank-line-between-blocks:
#         enabled: true
#         severity: warning
#   lint:
#     rules:
#       lint.terraform-required-version:
#         enabled: true
#         severity: error
`
}

// generateMonorepoConfig generates configuration for monorepos.
func generateMonorepoConfig() string {
	return `# TerraTidy Configuration for Monorepo
# Documentation: https://github.com/santosr2/TerraTidy
version: 1

# Global settings optimized for monorepos
severity_threshold: warning
fail_fast: false
parallel: true

# Engine configurations
engines:
  fmt:
    enabled: true

  style:
    enabled: true

  lint:
    enabled: true
    # Enable AWS/GCP/Azure plugins as needed
    # plugins:
    #   - aws

  policy:
    enabled: true
    # Central policies for the organization
    policy_dirs:
      - ./policies

# Profiles for different environments/teams
profiles:
  ci:
    description: "CI/CD pipeline checks"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: true

  development:
    description: "Local development settings"
    engines:
      fmt:
        enabled: true
      style:
        enabled: true
      lint:
        enabled: true
      policy:
        enabled: false

# Lint engine rule configuration
# engines:
#   lint:
#     rules:
#       # Enforce terraform version in all modules
#       lint.terraform-required-version:
#         enabled: true
#         severity: error
#       # Enforce provider version constraints
#       lint.terraform-required-providers:
#         enabled: true
#         severity: error
`
}

// generateCustomConfig generates a custom configuration based on user choices.
func generateCustomConfig(opts customConfigOptions) string {
	return fmt.Sprintf(`# TerraTidy Configuration
# Generated with interactive setup
version: 1

# Global settings
severity_threshold: %s
fail_fast: %t
parallel: true

# Engine configurations
engines:
  fmt:
    enabled: %t

  style:
    enabled: %t

  lint:
    enabled: %t

  policy:
    enabled: %t
`, opts.severity, opts.failFast, opts.fmtEnabled, opts.styleEnabled, opts.lintEnabled, opts.policyEnabled)
}
