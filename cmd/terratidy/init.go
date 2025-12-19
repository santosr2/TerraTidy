// Package main provides the init command for TerraTidy.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration")
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	configPath := ".terratidy.yaml"

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
	}

	fmt.Println("Initializing TerraTidy configuration...")
	fmt.Println()

	var config string

	if initInteractive {
		var err error
		config, err = interactiveInit()
		if err != nil {
			return fmt.Errorf("interactive setup failed: %w", err)
		}
	} else if initSplit {
		return initSplitConfig()
	} else if initMonorepo {
		config = generateMonorepoConfig()
	} else {
		config = generateDefaultConfig()
	}

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
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
	fmtEnabled := readYesNo(reader, true)

	fmt.Print("Enable style checks? [Y/n]: ")
	styleEnabled := readYesNo(reader, true)

	fmt.Print("Enable linting? [Y/n]: ")
	lintEnabled := readYesNo(reader, true)

	fmt.Print("Enable policy checks? [y/N]: ")
	policyEnabled := readYesNo(reader, false)

	// Ask about severity
	fmt.Print("Minimum severity threshold (info/warning/error) [warning]: ")
	severity := readLine(reader)
	if severity == "" {
		severity = "warning"
	}

	// Ask about fail-fast
	fmt.Print("Stop on first error? [y/N]: ")
	failFast := readYesNo(reader, false)

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

func readYesNo(reader *bufio.Reader, defaultYes bool) bool {
	line := readLine(reader)
	if line == "" {
		return defaultYes
	}
	return strings.ToLower(line) == "y" || strings.ToLower(line) == "yes"
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// initSplitConfig creates a modular split configuration.
func initSplitConfig() error {
	// Create .terratidy directory
	configDir := ".terratidy"
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
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
    config:
      indent_size: 2
`

	styleConfig := `# Style Configuration
engines:
  style:
    enabled: true

overrides:
  rules:
    # Customize style rules here
    # style.blank-lines-between-blocks:
    #   enabled: true
    #   severity: warning
`

	lintConfig := `# Linting Configuration
engines:
  lint:
    enabled: true
    config:
      # Enable specific plugins
      # plugins:
      #   - aws
      #   - google
`

	policyConfig := `# Policy Configuration
engines:
  policy:
    enabled: false  # Enable when you have policies defined
    config:
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
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
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
# Documentation: https://github.com/santosr2/terratidy
version: 1

# Global settings
severity_threshold: warning
fail_fast: false
parallel: true

# Engine configurations
engines:
  fmt:
    enabled: true
    config:
      indent_size: 2

  style:
    enabled: true
    config:
      # Style rules are enabled by default

  lint:
    enabled: true
    config:
      # Linting rules are enabled by default

  policy:
    enabled: false
    config:
      # Enable and configure policy directories
      # policy_dirs:
      #   - ./policies

# Override specific rules
# overrides:
#   rules:
#     style.blank-lines-between-blocks:
#       enabled: true
#       severity: warning
#     lint.terraform_required_version:
#       enabled: true
#       severity: error

# Custom rules (advanced)
# custom_rules:
#   my-custom-rule:
#     enabled: true
#     severity: warning
#     config:
#       option: value
`
}

// generateMonorepoConfig generates configuration for monorepos.
func generateMonorepoConfig() string {
	return `# TerraTidy Configuration for Monorepo
# Documentation: https://github.com/santosr2/terratidy
version: 1

# Global settings optimized for monorepos
severity_threshold: warning
fail_fast: false
parallel: true

# Engine configurations
engines:
  fmt:
    enabled: true
    config:
      indent_size: 2

  style:
    enabled: true
    config:
      # Monorepo-friendly settings

  lint:
    enabled: true
    config:
      # Enable AWS/GCP/Azure plugins as needed
      # plugins:
      #   - aws

  policy:
    enabled: true
    config:
      # Central policies for the organization
      policy_dirs:
        - ./policies

# Profiles for different environments/teams
profiles:
  ci:
    profile: ci
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
    profile: development
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

# Override rules globally
overrides:
  rules:
    # Enforce terraform version in all modules
    lint.terraform_required_version:
      enabled: true
      severity: error
    # Enforce provider version constraints
    lint.terraform_required_providers:
      enabled: true
      severity: error
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
    config:
      indent_size: 2

  style:
    enabled: %t

  lint:
    enabled: %t

  policy:
    enabled: %t
`, opts.severity, opts.failFast, opts.fmtEnabled, opts.styleEnabled, opts.lintEnabled, opts.policyEnabled)
}
