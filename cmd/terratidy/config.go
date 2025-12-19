// Package main provides configuration management commands for TerraTidy.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santosr2/terratidy/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configOutputFormat string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long: `Manage TerraTidy configuration files.

Use subcommands to show, validate, split, or merge configurations.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration",
	Long: `Display the final configuration after all imports and merges.

This command loads the configuration file, processes all imports,
applies profile settings, and shows the final resolved configuration.`,
	Example: `  # Show resolved config
  terratidy config show

  # Show config in JSON format
  terratidy config show --format json

  # Show specific config file
  terratidy config show --config custom.yaml`,
	RunE: runConfigShow,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the configuration file and all imports.

This command checks for syntax errors, invalid values, and missing
required fields in the configuration.`,
	Example: `  # Validate default config
  terratidy config validate

  # Validate specific config file
  terratidy config validate --config custom.yaml`,
	RunE: runConfigValidate,
}

var configSplitCmd = &cobra.Command{
	Use:   "split",
	Short: "Split configuration into modular structure",
	Long: `Convert a single .terratidy.yaml file into a modular directory structure.

This creates a .terratidy directory with separate files for each engine,
making it easier to manage complex configurations.`,
	Example: `  # Split default config
  terratidy config split

  # Split specific config file
  terratidy config split --config custom.yaml`,
	RunE: runConfigSplit,
}

var configMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge split configurations into single file",
	Long: `Combine modular configuration files into a single .terratidy.yaml.

This is useful for creating a distributable configuration or simplifying
a split configuration.`,
	Example: `  # Merge config files
  terratidy config merge

  # Merge and output to specific file
  terratidy config merge --output merged.yaml`,
	RunE: runConfigMerge,
}

var configInitProfileCmd = &cobra.Command{
	Use:   "init-profile [name]",
	Short: "Initialize a new configuration profile",
	Long:  `Create a new configuration profile in the config file.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Create a CI profile
  terratidy config init-profile ci

  # Create a development profile
  terratidy config init-profile dev`,
	RunE: runConfigInitProfile,
}

func init() {
	configShowCmd.Flags().StringVar(&configOutputFormat, "format", "yaml", "output format (yaml|json)")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configSplitCmd)
	configCmd.AddCommand(configMergeCmd)
	configCmd.AddCommand(configInitProfileCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var output []byte
	switch strings.ToLower(configOutputFormat) {
	case "json":
		output, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml":
		output, err = yaml.Marshal(cfg)
	default:
		return fmt.Errorf("unsupported format: %s (use yaml or json)", configOutputFormat)
	}

	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

func runConfigValidate(_ *cobra.Command, _ []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = ".terratidy.yaml"
	}

	fmt.Printf("Validating configuration: %s\n\n", configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Load and validate
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Println("[!] Validation failed:")
		fmt.Printf("    %v\n", err)
		return err
	}

	// Additional validation
	issues := validateConfig(cfg)

	if len(issues) > 0 {
		fmt.Println("[!] Validation warnings:")
		for _, issue := range issues {
			fmt.Printf("    - %s\n", issue)
		}
		fmt.Println()
	}

	fmt.Println("[+] Configuration is valid")

	// Show summary
	fmt.Println()
	fmt.Println("Configuration summary:")
	fmt.Printf("  Version: %d\n", cfg.Version)
	fmt.Printf("  Engines enabled:\n")
	if cfg.Engines.Fmt.Enabled {
		fmt.Println("    - fmt")
	}
	if cfg.Engines.Style.Enabled {
		fmt.Println("    - style")
	}
	if cfg.Engines.Lint.Enabled {
		fmt.Println("    - lint")
	}
	if cfg.Engines.Policy.Enabled {
		fmt.Println("    - policy")
	}

	if len(cfg.Profiles) > 0 {
		fmt.Printf("  Profiles: %d\n", len(cfg.Profiles))
		for name := range cfg.Profiles {
			fmt.Printf("    - %s\n", name)
		}
	}

	return nil
}

// validateConfig performs additional validation on the configuration.
func validateConfig(cfg *config.Config) []string {
	var issues []string

	// Check severity threshold
	if cfg.SeverityThreshold != "" {
		validSeverities := map[string]bool{"info": true, "warning": true, "error": true}
		if !validSeverities[strings.ToLower(cfg.SeverityThreshold)] {
			msg := fmt.Sprintf(
				"invalid severity_threshold: %s (use info, warning, or error)",
				cfg.SeverityThreshold,
			)
			issues = append(issues, msg)
		}
	}

	// Check for empty engines
	if !cfg.Engines.Fmt.Enabled && !cfg.Engines.Style.Enabled &&
		!cfg.Engines.Lint.Enabled && !cfg.Engines.Policy.Enabled {
		issues = append(issues, "no engines are enabled")
	}

	return issues
}

func runConfigSplit(_ *cobra.Command, _ []string) error {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configPath)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	configDir := ".terratidy"
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	fmt.Println("Splitting configuration...")

	if err := writeEngineConfigs(cfg, configDir); err != nil {
		return err
	}

	if err := writeMainConfig(configPath, cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Configuration split complete!")
	return nil
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return ".terratidy.yaml"
}

func writeEngineConfigs(cfg *config.Config, configDir string) error {
	engines := []struct {
		name    string
		enabled bool
		config  interface{}
	}{
		{"fmt", cfg.Engines.Fmt.Enabled, cfg.Engines.Fmt},
		{"style", cfg.Engines.Style.Enabled, cfg.Engines.Style},
		{"lint", cfg.Engines.Lint.Enabled, cfg.Engines.Lint},
		{"policy", cfg.Engines.Policy.Enabled, cfg.Engines.Policy},
	}

	for _, eng := range engines {
		if !eng.enabled {
			continue
		}
		if err := writeEngineConfig(configDir, eng.name, eng.config); err != nil {
			return err
		}
	}
	return nil
}

func writeEngineConfig(configDir, name string, engineCfg interface{}) error {
	cfgMap := map[string]interface{}{
		"engines": map[string]interface{}{
			name: engineCfg,
		},
	}
	filePath := filepath.Join(configDir, name+".yaml")
	if err := writeYAMLFile(filePath, cfgMap); err != nil {
		return err
	}
	fmt.Printf("  Created %s\n", filePath)
	return nil
}

func writeMainConfig(configPath string, cfg *config.Config) error {
	mainCfg := fmt.Sprintf(`# TerraTidy Configuration
# Split configuration - engine settings are in .terratidy/
version: %d

imports:
  - ".terratidy/*.yaml"

# Global settings
severity_threshold: %s
fail_fast: %t
parallel: %t
`, cfg.Version, cfg.SeverityThreshold, cfg.FailFast, cfg.Parallel)

	if err := os.WriteFile(configPath, []byte(mainCfg), 0o644); err != nil {
		return fmt.Errorf("writing main config: %w", err)
	}
	fmt.Printf("  Updated %s\n", configPath)
	return nil
}

func runConfigMerge(_ *cobra.Command, _ []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = ".terratidy.yaml"
	}

	fmt.Println("Merging configurations...")

	// Load and resolve all imports
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Clear imports since we're merging
	cfg.Imports = nil

	// Write merged config
	output, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Add header comment
	header := `# TerraTidy Configuration
# Merged from split configuration
`
	finalOutput := header + string(output)

	if err := os.WriteFile(configPath, []byte(finalOutput), 0o644); err != nil {
		return fmt.Errorf("writing merged config: %w", err)
	}

	fmt.Printf("Merged configuration written to %s\n", configPath)
	return nil
}

func runConfigInitProfile(_ *cobra.Command, args []string) error {
	profileName := args[0]

	configPath := cfgFile
	if configPath == "" {
		configPath = ".terratidy.yaml"
	}

	// Load current config
	cfg, err := config.Load(configPath)
	if err != nil {
		// Create new config if doesn't exist
		cfg = config.DefaultConfig()
	}

	// Check if profile already exists
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.Profile)
	}

	if _, exists := cfg.Profiles[profileName]; exists {
		return fmt.Errorf("profile '%s' already exists", profileName)
	}

	// Create new profile
	cfg.Profiles[profileName] = config.Profile{
		Name:        profileName,
		Description: fmt.Sprintf("%s profile", profileName),
		Engines: config.Engines{
			Fmt:    config.EngineConfig{Enabled: true},
			Style:  config.EngineConfig{Enabled: true},
			Lint:   config.EngineConfig{Enabled: true},
			Policy: config.EngineConfig{Enabled: false},
		},
	}

	// Write updated config
	output, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Created profile '%s' in %s\n", profileName, configPath)
	fmt.Println()
	fmt.Printf("Use it with: terratidy check --profile %s\n", profileName)

	return nil
}

// writeYAMLFile writes data to a YAML file.
func writeYAMLFile(path string, data interface{}) error {
	output, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", path, err)
	}
	return os.WriteFile(path, output, 0o644)
}
