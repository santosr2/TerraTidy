package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// envVarPattern matches ${VAR} or ${VAR:-default} patterns
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Config represents the complete TerraTidy configuration
type Config struct {
	Version  int                `yaml:"version"`
	Imports  []string           `yaml:"imports,omitempty"`
	Engines  Engines            `yaml:"engines"`
	Profiles map[string]Profile `yaml:"profiles,omitempty"`

	// Global settings
	SeverityThreshold string `yaml:"severity_threshold,omitempty"`
	FailFast          bool   `yaml:"fail_fast,omitempty"`
	Parallel          bool   `yaml:"parallel,omitempty"`

	// Overrides
	Overrides OverridesConfig `yaml:"overrides,omitempty"`

	// Plugin settings
	Plugins PluginsConfig `yaml:"plugins,omitempty"`

	// Custom rules
	CustomRules map[string]RuleConfig `yaml:"custom_rules,omitempty"`
}

// Engines configuration for each engine
type Engines struct {
	Fmt    EngineConfig `yaml:"fmt"`
	Style  EngineConfig `yaml:"style"`
	Lint   EngineConfig `yaml:"lint"`
	Policy EngineConfig `yaml:"policy"`
}

// EngineConfig represents configuration for a single engine
type EngineConfig struct {
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config,omitempty"`
}

// Profile represents a configuration profile
type Profile struct {
	Name            string          `yaml:"profile"`
	Description     string          `yaml:"description"`
	Inherits        string          `yaml:"inherits,omitempty"`
	Engines         Engines         `yaml:"engines"`
	DisabledEngines []string        `yaml:"disabled_engines,omitempty"` // Explicitly disable inherited engines
	Overrides       OverridesConfig `yaml:"overrides,omitempty"`
}

// OverridesConfig allows overriding specific settings
type OverridesConfig struct {
	Rules map[string]RuleConfig `yaml:"rules,omitempty"`
}

// RuleConfig represents configuration for a single rule
type RuleConfig struct {
	Enabled  bool                   `yaml:"enabled"`
	Severity string                 `yaml:"severity,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty"`
}

// PluginsConfig represents plugin settings
type PluginsConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Directories []string `yaml:"directories,omitempty"`
}

// Load loads the configuration from the specified path
func Load(path string) (*Config, error) {
	if path == "" {
		path = ".terratidy.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	// Expand environment variables in the config
	expandedData := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Load imports if specified
	if len(cfg.Imports) > 0 {
		if err := cfg.loadImports(filepath.Dir(path)); err != nil {
			return nil, fmt.Errorf("loading imports: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// expandEnvVars expands environment variables in the config content
// Supports ${VAR} and ${VAR:-default} syntax
func expandEnvVars(content string) string {
	return envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the variable expression (without ${ and })
		expr := match[2 : len(match)-1]

		// Check for default value syntax: VAR:-default
		if idx := strings.Index(expr, ":-"); idx != -1 {
			varName := expr[:idx]
			defaultVal := expr[idx+2:]

			if val := os.Getenv(varName); val != "" {
				return val
			}
			return defaultVal
		}

		// Check for required syntax: VAR:?error message
		if idx := strings.Index(expr, ":?"); idx != -1 {
			varName := expr[:idx]
			// Return the variable value or keep the placeholder (validation will catch it)
			if val := os.Getenv(varName); val != "" {
				return val
			}
			// Return empty for now; validation can catch undefined required vars
			return ""
		}

		// Simple variable: ${VAR}
		return os.Getenv(expr)
	})
}

// loadImports loads and merges imported configurations
func (c *Config) loadImports(baseDir string) error {
	for _, pattern := range c.Imports {
		// Convert relative pattern to absolute
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}

		// Expand glob pattern
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid import pattern %s: %w", pattern, err)
		}

		// Load each matched file
		for _, match := range matches {
			partial, err := loadPartialConfig(match)
			if err != nil {
				return fmt.Errorf("loading %s: %w", match, err)
			}

			// Merge partial config into main config
			c.merge(partial)
		}
	}

	return nil
}

// loadPartialConfig loads a partial configuration file
func loadPartialConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// merge merges another config into this one
func (c *Config) merge(other *Config) {
	// Merge custom rules
	if c.CustomRules == nil {
		c.CustomRules = make(map[string]RuleConfig)
	}
	for k, v := range other.CustomRules {
		c.CustomRules[k] = v
	}

	// Merge override rules
	if c.Overrides.Rules == nil {
		c.Overrides.Rules = make(map[string]RuleConfig)
	}
	for k, v := range other.Overrides.Rules {
		c.Overrides.Rules[k] = v
	}

	// Merge profiles
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	for k, v := range other.Profiles {
		c.Profiles[k] = v
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == 0 {
		c.Version = 1
	}

	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", c.Version)
	}

	// Validate severity threshold
	if c.SeverityThreshold != "" {
		validSeverities := map[string]bool{
			"error":   true,
			"warning": true,
			"info":    true,
		}
		if !validSeverities[c.SeverityThreshold] {
			return fmt.Errorf("invalid severity_threshold: %s (must be error, warning, or info)", c.SeverityThreshold)
		}
	}

	// Validate profiles
	if err := c.validateProfiles(); err != nil {
		return fmt.Errorf("profile validation: %w", err)
	}

	// Validate custom rules
	if err := c.validateCustomRules(); err != nil {
		return fmt.Errorf("custom rules validation: %w", err)
	}

	// Validate plugin configuration
	if err := c.validatePlugins(); err != nil {
		return fmt.Errorf("plugins validation: %w", err)
	}

	return nil
}

// validateProfiles validates profile configurations
func (c *Config) validateProfiles() error {
	// Check for circular inheritance
	for name, profile := range c.Profiles {
		if profile.Inherits != "" {
			if err := c.checkCircularInheritance(name, make(map[string]bool)); err != nil {
				return err
			}

			// Check that inherited profile exists
			if _, exists := c.Profiles[profile.Inherits]; !exists {
				return fmt.Errorf("profile %q inherits from non-existent profile %q", name, profile.Inherits)
			}
		}
	}

	return nil
}

// checkCircularInheritance checks for circular profile inheritance
func (c *Config) checkCircularInheritance(name string, visited map[string]bool) error {
	if visited[name] {
		return fmt.Errorf("circular inheritance detected involving profile %q", name)
	}

	visited[name] = true

	profile, exists := c.Profiles[name]
	if !exists {
		return nil
	}

	if profile.Inherits != "" {
		return c.checkCircularInheritance(profile.Inherits, visited)
	}

	return nil
}

// validateCustomRules validates custom rule configurations
func (c *Config) validateCustomRules() error {
	validSeverities := map[string]bool{
		"error":   true,
		"warning": true,
		"info":    true,
		"":        true, // Allow empty (default)
	}

	for name, rule := range c.CustomRules {
		if name == "" {
			return fmt.Errorf("custom rule name cannot be empty")
		}

		if !validSeverities[rule.Severity] {
			return fmt.Errorf("custom rule %q has invalid severity: %s", name, rule.Severity)
		}
	}

	// Also validate override rules
	for name, rule := range c.Overrides.Rules {
		if name == "" {
			return fmt.Errorf("override rule name cannot be empty")
		}

		if !validSeverities[rule.Severity] {
			return fmt.Errorf("override rule %q has invalid severity: %s", name, rule.Severity)
		}
	}

	return nil
}

// validatePlugins validates plugin configuration
func (c *Config) validatePlugins() error {
	if !c.Plugins.Enabled {
		return nil
	}

	for _, dir := range c.Plugins.Directories {
		if dir == "" {
			return fmt.Errorf("plugin directory cannot be empty")
		}

		// Check if directory exists (optional - warn only if enabled but directory missing)
		// We don't fail here as plugins might be optional
	}

	return nil
}

// GetProfile returns a profile with all inherited settings resolved
func (c *Config) GetProfile(name string) (*Profile, error) {
	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile %q not found", name)
	}

	// If no inheritance, return as-is
	if profile.Inherits == "" {
		return &profile, nil
	}

	// Resolve inheritance chain
	resolved, err := c.resolveProfileInheritance(name, make(map[string]bool))
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

// resolveProfileInheritance resolves a profile with all inherited settings
func (c *Config) resolveProfileInheritance(name string, visited map[string]bool) (*Profile, error) {
	if visited[name] {
		return nil, fmt.Errorf("circular inheritance detected involving profile %q", name)
	}
	visited[name] = true

	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile %q not found", name)
	}

	// If no inheritance, return a copy
	if profile.Inherits == "" {
		result := profile // Copy
		return &result, nil
	}

	// Get parent profile first
	parent, err := c.resolveProfileInheritance(profile.Inherits, visited)
	if err != nil {
		return nil, err
	}

	// Merge: child settings override parent
	resolved := c.mergeProfiles(parent, &profile)
	return resolved, nil
}

// mergeProfiles merges a child profile into a parent, with child taking precedence
// Note: Due to YAML parsing limitations, child profiles can only add/configure engines.
// To disable an engine, use disabled_engines in the profile.
func (c *Config) mergeProfiles(parent, child *Profile) *Profile {
	result := &Profile{
		Name:        child.Name,
		Description: child.Description,
		Inherits:    child.Inherits,
	}

	// If child has a description, use it; otherwise inherit
	if result.Description == "" && parent != nil {
		result.Description = parent.Description
	}

	// Merge engines - start with parent engines
	if parent != nil {
		result.Engines = parent.Engines
	}

	// Child engines ADD to parent - if child explicitly enables or configures an engine, use it
	// Only override parent if child has something explicit (Enabled=true or Config present)
	if child.Engines.Fmt.Enabled || len(child.Engines.Fmt.Config) > 0 {
		result.Engines.Fmt = child.Engines.Fmt
	}
	if child.Engines.Style.Enabled || len(child.Engines.Style.Config) > 0 {
		result.Engines.Style = child.Engines.Style
	}
	if child.Engines.Lint.Enabled || len(child.Engines.Lint.Config) > 0 {
		result.Engines.Lint = child.Engines.Lint
	}
	if child.Engines.Policy.Enabled || len(child.Engines.Policy.Config) > 0 {
		result.Engines.Policy = child.Engines.Policy
	}

	// Apply disabled_engines from child (explicit disables)
	for _, engineName := range child.DisabledEngines {
		switch engineName {
		case "fmt":
			result.Engines.Fmt.Enabled = false
		case "style":
			result.Engines.Style.Enabled = false
		case "lint":
			result.Engines.Lint.Enabled = false
		case "policy":
			result.Engines.Policy.Enabled = false
		}
	}

	// Merge overrides - child overrides win
	result.Overrides.Rules = make(map[string]RuleConfig)
	if parent != nil {
		for k, v := range parent.Overrides.Rules {
			result.Overrides.Rules[k] = v
		}
	}
	for k, v := range child.Overrides.Rules {
		result.Overrides.Rules[k] = v
	}

	return result
}

// ApplyProfile applies a profile's settings to the config
func (c *Config) ApplyProfile(name string) error {
	profile, err := c.GetProfile(name)
	if err != nil {
		return err
	}

	// Apply engine settings from profile
	c.Engines = profile.Engines

	// Merge overrides
	if c.Overrides.Rules == nil {
		c.Overrides.Rules = make(map[string]RuleConfig)
	}
	for k, v := range profile.Overrides.Rules {
		c.Overrides.Rules[k] = v
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Engines: Engines{
			Fmt:    EngineConfig{Enabled: true},
			Style:  EngineConfig{Enabled: true},
			Lint:   EngineConfig{Enabled: true},
			Policy: EngineConfig{Enabled: false}, // Opt-in
		},
		SeverityThreshold: "warning",
		FailFast:          false,
		Parallel:          true,
	}
}
