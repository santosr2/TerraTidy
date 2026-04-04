// Package config provides configuration loading and validation for TerraTidy.
// It supports YAML configuration files with environment variable expansion,
// profile inheritance, and custom rule configuration.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Security limits for glob operations
const (
	// maxImportGlobResults is the maximum number of files a single import pattern can match.
	// This prevents resource exhaustion from overly broad glob patterns.
	maxImportGlobResults = 1000

	// globTimeout is the maximum time allowed for a single glob operation.
	// This prevents hangs from complex patterns on large filesystems.
	globTimeout = 5 * time.Second
)

// envVarPattern matches ${VAR} or ${VAR:-default} patterns
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// sensitiveVarPatterns contains substrings that indicate a potentially sensitive variable.
// These are checked case-insensitively against variable names during expansion.
var sensitiveVarPatterns = []string{
	"SECRET",
	"PASSWORD",
	"TOKEN",
	"KEY",
	"CREDENTIAL",
	"PRIVATE",
}

// configLogger is the logger used for config-related warnings.
// It writes to stderr with a prefix for easy identification.
var configLogger = log.New(os.Stderr, "terratidy-config: ", log.LstdFlags)

// isSensitiveVar checks if a variable name contains sensitive patterns.
func isSensitiveVar(varName string) bool {
	upper := strings.ToUpper(varName)
	for _, pattern := range sensitiveVarPatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

// warnIfSensitive logs a warning if the variable name appears sensitive.
// The warning never includes the actual value, only the variable name.
func warnIfSensitive(varName string) {
	if isSensitiveVar(varName) {
		configLogger.Printf("[WARN] expanding potentially sensitive variable: ${%s}", varName)
	}
}

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
	Enabled *bool          `yaml:"enabled,omitempty"`
	Config  map[string]any `yaml:"config,omitempty"`
}

// IsEnabled returns whether the engine is enabled.
// Returns false if Enabled is nil (not set).
func (e EngineConfig) IsEnabled() bool {
	if e.Enabled == nil {
		return false
	}
	return *e.Enabled
}

// BoolPtr returns a pointer to the given bool value.
// Use this when setting EngineConfig.Enabled explicitly.
func BoolPtr(b bool) *bool {
	return &b
}

// Profile represents a configuration profile
type Profile struct {
	Name        string          `yaml:"profile"`
	Description string          `yaml:"description"`
	Inherits    string          `yaml:"inherits,omitempty"`
	Engines     Engines         `yaml:"engines"`
	Overrides   OverridesConfig `yaml:"overrides,omitempty"`
}

// OverridesConfig allows overriding specific settings
type OverridesConfig struct {
	Rules map[string]RuleConfig `yaml:"rules,omitempty"`
}

// RuleConfig represents configuration for a single rule
type RuleConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Severity string         `yaml:"severity,omitempty"`
	Config   map[string]any `yaml:"config,omitempty"`
}

// PluginsConfig represents plugin settings
type PluginsConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Directories     []string `yaml:"directories,omitempty"`
	VerifyIntegrity *bool    `yaml:"verify_integrity,omitempty"`
}

// ShouldVerifyIntegrity returns whether plugin integrity verification is enabled.
// Defaults to true if not explicitly set (secure by default).
func (p *PluginsConfig) ShouldVerifyIntegrity() bool {
	if p.VerifyIntegrity == nil {
		return true // Secure by default
	}
	return *p.VerifyIntegrity
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
		absPath, _ := filepath.Abs(path)
		visited := map[string]bool{absPath: true}
		if err := cfg.loadImports(filepath.Dir(path), visited); err != nil {
			return nil, fmt.Errorf("loading imports: %w", err)
		}
	}

	// Apply defaults and validate
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// expandEnvVars expands environment variables in the config content
// Supports ${VAR} and ${VAR:-default} syntax
// Logs a warning (to stderr) when expanding variables with sensitive-looking names.
func expandEnvVars(content string) string {
	return envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the variable expression (without ${ and })
		expr := match[2 : len(match)-1]

		// Check for default value syntax: VAR:-default
		if idx := strings.Index(expr, ":-"); idx != -1 {
			varName := expr[:idx]
			defaultVal := expr[idx+2:]

			if val := os.Getenv(varName); val != "" {
				warnIfSensitive(varName)
				return val
			}
			return defaultVal
		}

		// Check for required syntax: VAR:?error message
		if idx := strings.Index(expr, ":?"); idx != -1 {
			varName := expr[:idx]
			// Return the variable value or keep the placeholder (validation will catch it)
			if val := os.Getenv(varName); val != "" {
				warnIfSensitive(varName)
				return val
			}
			// Return empty for now; validation can catch undefined required vars
			return ""
		}

		// Simple variable: ${VAR}
		warnIfSensitive(expr)
		return os.Getenv(expr)
	})
}

// globWithTimeout executes filepath.Glob with a timeout to prevent hangs on complex patterns.
func globWithTimeout(pattern string, timeout time.Duration) ([]string, error) {
	type result struct {
		matches []string
		err     error
	}

	ch := make(chan result, 1)
	go func() {
		matches, err := filepath.Glob(pattern)
		ch <- result{matches, err}
	}()

	select {
	case res := <-ch:
		return res.matches, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("glob pattern %q timed out after %v", pattern, timeout)
	}
}

// loadImports loads and merges imported configurations.
// The visited map tracks already-loaded files to detect circular imports.
func (c *Config) loadImports(baseDir string, visited map[string]bool) error {
	for _, pattern := range c.Imports {
		// Convert relative pattern to absolute
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}

		// Expand glob pattern with timeout
		matches, err := globWithTimeout(pattern, globTimeout)
		if err != nil {
			return fmt.Errorf("expanding import pattern %s: %w", pattern, err)
		}

		// Check result count to prevent resource exhaustion
		if len(matches) > maxImportGlobResults {
			return fmt.Errorf("import pattern %q matched too many files (%d > %d)", pattern, len(matches), maxImportGlobResults)
		}

		// Load each matched file
		for _, match := range matches {
			absMatch, _ := filepath.Abs(match)
			if visited[absMatch] {
				return fmt.Errorf("circular import detected: %s", absMatch)
			}
			visited[absMatch] = true

			partial, err := loadPartialConfig(match)
			if err != nil {
				return fmt.Errorf("loading %s: %w", match, err)
			}

			// Recursively load imports from the partial config
			if len(partial.Imports) > 0 {
				if err := partial.loadImports(filepath.Dir(match), visited); err != nil {
					return err
				}
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
	for k := range other.Profiles {
		c.Profiles[k] = other.Profiles[k]
	}
}

// SetDefaults fills in default values for unset fields.
// Call this before Validate.
func (c *Config) SetDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
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
	for name := range c.Profiles {
		profile := c.Profiles[name]
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

// mergeProfiles merges a child profile into a parent, with child taking precedence.
// Child profile can override any engine setting from parent using engines.<name>.enabled.
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

	// Child engines override parent if explicitly set (Enabled != nil or Config present)
	if child.Engines.Fmt.Enabled != nil || len(child.Engines.Fmt.Config) > 0 {
		result.Engines.Fmt = child.Engines.Fmt
	}
	if child.Engines.Style.Enabled != nil || len(child.Engines.Style.Config) > 0 {
		result.Engines.Style = child.Engines.Style
	}
	if child.Engines.Lint.Enabled != nil || len(child.Engines.Lint.Config) > 0 {
		result.Engines.Lint = child.Engines.Lint
	}
	if child.Engines.Policy.Enabled != nil || len(child.Engines.Policy.Config) > 0 {
		result.Engines.Policy = child.Engines.Policy
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
			Fmt:    EngineConfig{Enabled: BoolPtr(true)},
			Style:  EngineConfig{Enabled: BoolPtr(true)},
			Lint:   EngineConfig{Enabled: BoolPtr(true)},
			Policy: EngineConfig{Enabled: BoolPtr(false)}, // Opt-in
		},
		SeverityThreshold: "warning",
		FailFast:          false,
		Parallel:          true,
	}
}
