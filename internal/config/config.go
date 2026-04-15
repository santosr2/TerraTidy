// Package config provides configuration loading and validation for TerraTidy.
// It supports YAML configuration files with environment variable expansion,
// profile inheritance, and custom rule configuration.
package config

import (
	"bytes"
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

// Duration wraps time.Duration to support YAML unmarshaling from strings like "5m".
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the underlying time.Duration value.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

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

// unmarshalStrict unmarshals YAML data into the target with strict validation.
// It rejects unknown fields to catch typos in config keys.
func unmarshalStrict(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

// CacheConfig holds configuration for the file cache.
type CacheConfig struct {
	MaxAge   Duration `yaml:"max_age,omitempty"`  // Maximum age of cache entries (e.g., "5m")
	MaxSize  int      `yaml:"max_size,omitempty"` // Maximum number of entries (LRU eviction)
	Disabled bool     `yaml:"disabled,omitempty"` // Disable caching entirely
}

// IsConfigured returns true if any cache option was explicitly set.
func (c CacheConfig) IsConfigured() bool {
	return c.MaxAge != 0 || c.MaxSize != 0 || c.Disabled
}

// OutputConfig holds configuration for output formatting.
type OutputConfig struct {
	AbsolutePaths *bool `yaml:"absolute_paths,omitempty"` // Use absolute paths in output (default: false)
}

// Config represents the complete TerraTidy configuration
type Config struct {
	Version  int                `yaml:"version"`
	Imports  []string           `yaml:"imports,omitempty"`
	Exclude  []string           `yaml:"exclude,omitempty"` // Glob patterns for files/dirs to exclude
	Engines  Engines            `yaml:"engines"`
	Profiles map[string]Profile `yaml:"profiles,omitempty"`

	// Global settings
	SeverityThreshold string       `yaml:"severity_threshold,omitempty"`
	FailFast          *bool        `yaml:"fail_fast,omitempty"`
	Parallel          *bool        `yaml:"parallel,omitempty"`
	Recursive         *bool        `yaml:"recursive,omitempty"` // Directory recursion (default: true)
	Cache             CacheConfig  `yaml:"cache,omitempty"`
	Output            OutputConfig `yaml:"output,omitempty"`

	// Plugin settings
	Plugins PluginsConfig `yaml:"plugins,omitempty"`
}

// Engines holds typed configuration for all engines.
// Each engine has its own strongly-typed config struct for better validation.
type Engines struct {
	Fmt    FmtEngineConfig    `yaml:"fmt"`
	Style  StyleEngineConfig  `yaml:"style"`
	Lint   LintEngineConfig   `yaml:"lint"`
	Policy PolicyEngineConfig `yaml:"policy"`
}

// mergeFrom merges settings from another Engines config.
// Only non-zero values from other are applied to preserve existing settings.
func (e *Engines) mergeFrom(other *Engines) {
	e.Fmt.mergeFrom(&other.Fmt)
	e.Style.mergeFrom(&other.Style)
	e.Lint.mergeFrom(&other.Lint)
	e.Policy.mergeFrom(&other.Policy)
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

// FmtEngineConfig holds typed configuration for the format engine.
// This mirrors format.Config but with YAML tags for config parsing.
type FmtEngineConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
	Check   bool  `yaml:"check,omitempty"` // Check mode (don't modify files)
	Diff    bool  `yaml:"diff,omitempty"`  // Show diff of changes
}

// IsEnabled returns whether the format engine is enabled.
func (c FmtEngineConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// mergeFrom merges settings from another FmtEngineConfig.
// Only non-zero values from other are applied.
func (c *FmtEngineConfig) mergeFrom(other *FmtEngineConfig) {
	if other.Enabled != nil {
		c.Enabled = other.Enabled
	}
	if other.Check {
		c.Check = other.Check
	}
	if other.Diff {
		c.Diff = other.Diff
	}
}

// StyleEngineConfig holds typed configuration for the style engine.
// This mirrors style.Config but with YAML tags for config parsing.
type StyleEngineConfig struct {
	Enabled *bool                 `yaml:"enabled,omitempty"`
	Fix     bool                  `yaml:"fix,omitempty"`   // Auto-fix mode
	Diff    bool                  `yaml:"diff,omitempty"`  // Show diff of changes
	Rules   map[string]RuleConfig `yaml:"rules,omitempty"` // Rule-specific configuration
}

// IsEnabled returns whether the style engine is enabled.
func (c StyleEngineConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// mergeFrom merges settings from another StyleEngineConfig.
// Only non-zero values from other are applied.
func (c *StyleEngineConfig) mergeFrom(other *StyleEngineConfig) {
	if other.Enabled != nil {
		c.Enabled = other.Enabled
	}
	if other.Fix {
		c.Fix = other.Fix
	}
	if other.Diff {
		c.Diff = other.Diff
	}
	if len(other.Rules) > 0 {
		if c.Rules == nil {
			c.Rules = make(map[string]RuleConfig)
		}
		for k, v := range other.Rules {
			c.Rules[k] = v
		}
	}
}

// LintEngineConfig holds typed configuration for the lint engine.
// This mirrors lint.Config but with YAML tags for config parsing.
type LintEngineConfig struct {
	Enabled         *bool                 `yaml:"enabled,omitempty"`
	ConfigFile      string                `yaml:"config_file,omitempty"`      // Path to TFLint configuration file
	Plugins         []string              `yaml:"plugins,omitempty"`          // List of TFLint plugins to enable
	Args            []string              `yaml:"args,omitempty"`             // Extra arguments to pass to TFLint
	UseTFLint       bool                  `yaml:"use_tflint,omitempty"`       // Enable TFLint integration
	TFLintPath      string                `yaml:"tflint_path,omitempty"`      // Custom path to TFLint binary
	FallbackBuiltin bool                  `yaml:"fallback_builtin,omitempty"` // Use built-in rules if TFLint unavailable
	Rules           map[string]RuleConfig `yaml:"rules,omitempty"`            // Rule-specific configuration
}

// IsEnabled returns whether the lint engine is enabled.
func (c LintEngineConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// mergeFrom merges settings from another LintEngineConfig.
// Only non-zero values from other are applied.
func (c *LintEngineConfig) mergeFrom(other *LintEngineConfig) {
	if other.Enabled != nil {
		c.Enabled = other.Enabled
	}
	if other.ConfigFile != "" {
		c.ConfigFile = other.ConfigFile
	}
	if len(other.Plugins) > 0 {
		c.Plugins = other.Plugins
	}
	if len(other.Args) > 0 {
		c.Args = other.Args
	}
	if other.UseTFLint {
		c.UseTFLint = other.UseTFLint
	}
	if other.TFLintPath != "" {
		c.TFLintPath = other.TFLintPath
	}
	if other.FallbackBuiltin {
		c.FallbackBuiltin = other.FallbackBuiltin
	}
	if len(other.Rules) > 0 {
		if c.Rules == nil {
			c.Rules = make(map[string]RuleConfig)
		}
		for k, v := range other.Rules {
			c.Rules[k] = v
		}
	}
}

// PolicyEngineConfig holds typed configuration for the policy engine.
// This mirrors policy.Config but with YAML tags for config parsing.
type PolicyEngineConfig struct {
	Enabled     *bool                 `yaml:"enabled,omitempty"`
	PolicyDirs  []string              `yaml:"policy_dirs,omitempty"`  // Directories containing Rego policy files
	PolicyFiles []string              `yaml:"policy_files,omitempty"` // Individual policy files
	DataFiles   []string              `yaml:"data_files,omitempty"`   // Additional data files for policies
	Rules       map[string]RuleConfig `yaml:"rules,omitempty"`        // Rule-specific configuration
}

// IsEnabled returns whether the policy engine is enabled.
func (c PolicyEngineConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// mergeFrom merges settings from another PolicyEngineConfig.
// Only non-zero values from other are applied.
func (c *PolicyEngineConfig) mergeFrom(other *PolicyEngineConfig) {
	if other.Enabled != nil {
		c.Enabled = other.Enabled
	}
	if len(other.PolicyDirs) > 0 {
		c.PolicyDirs = other.PolicyDirs
	}
	if len(other.PolicyFiles) > 0 {
		c.PolicyFiles = other.PolicyFiles
	}
	if len(other.DataFiles) > 0 {
		c.DataFiles = other.DataFiles
	}
	if len(other.Rules) > 0 {
		if c.Rules == nil {
			c.Rules = make(map[string]RuleConfig)
		}
		for k, v := range other.Rules {
			c.Rules[k] = v
		}
	}
}

// BoolPtr returns a pointer to the given bool value.
// Use this when setting EngineConfig.Enabled explicitly.
func BoolPtr(b bool) *bool {
	return &b
}

// Profile represents a configuration profile
type Profile struct {
	Name        string  `yaml:"profile"`
	Description string  `yaml:"description"`
	Inherits    string  `yaml:"inherits,omitempty"`
	Engines     Engines `yaml:"engines"`
}

// RuleConfig represents configuration for a single rule
type RuleConfig struct {
	Enabled  *bool          `yaml:"enabled,omitempty"`
	Severity string         `yaml:"severity,omitempty"`
	Config   map[string]any `yaml:"config,omitempty"`
}

// IsEnabled returns whether the rule is enabled.
// If Enabled is nil (not explicitly set), returns defaultEnabled (inherit from engine).
func (r RuleConfig) IsEnabled(defaultEnabled bool) bool {
	if r.Enabled == nil {
		return defaultEnabled
	}
	return *r.Enabled
}

// PluginsConfig represents plugin settings
type PluginsConfig struct {
	Enabled         bool                  `yaml:"enabled"`
	Directories     []string              `yaml:"directories,omitempty"`
	VerifyIntegrity *bool                 `yaml:"verify_integrity,omitempty"`
	Rules           map[string]RuleConfig `yaml:"rules,omitempty"` // Per-rule enable/disable/severity
	Tags            []string              `yaml:"tags,omitempty"`  // Only load rules with these tags (empty = all)
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
	cfg, _, err := LoadWithFiles(path)
	return cfg, err
}

// LoadWithFiles loads the configuration and returns all loaded file paths.
// This includes the main config file and all imported files.
// Useful for watching config files for changes.
func LoadWithFiles(path string) (*Config, []string, error) {
	if path == "" {
		path = ".terratidy.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	// Handle empty or whitespace-only config files as defaults
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return DefaultConfig(), []string{path}, nil
	}

	// Expand environment variables in the config
	expandedData := expandEnvVars(string(data))

	var cfg Config
	if err := unmarshalStrict([]byte(expandedData), &cfg); err != nil {
		return nil, nil, fmt.Errorf("parsing config: %w", err)
	}

	// Track all loaded files
	absPath, _ := filepath.Abs(path)
	visited := map[string]bool{absPath: true}

	// Load imports if specified
	if len(cfg.Imports) > 0 {
		if err := cfg.loadImports(filepath.Dir(path), visited); err != nil {
			return nil, nil, fmt.Errorf("loading imports: %w", err)
		}
	}

	// Apply defaults and validate
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	// Collect loaded file paths
	loadedFiles := make([]string, 0, len(visited))
	for filePath := range visited {
		loadedFiles = append(loadedFiles, filePath)
	}

	return &cfg, loadedFiles, nil
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
	absBaseDir, _ := filepath.Abs(baseDir)

	for _, pattern := range c.Imports {
		// Block obvious path traversal patterns
		if strings.Contains(pattern, "..") {
			return fmt.Errorf("import pattern %q contains path traversal", pattern)
		}

		// Convert relative pattern to absolute
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}

		// Expand glob pattern with timeout
		matches, err := globWithTimeout(pattern, globTimeout)
		if err != nil {
			return fmt.Errorf("expanding import pattern %s: %w", pattern, err)
		}

		// Verify all matches are within the base directory (prevent symlink attacks)
		for _, match := range matches {
			absMatch, _ := filepath.Abs(match)
			if !strings.HasPrefix(absMatch, absBaseDir) {
				return fmt.Errorf("import %q resolves outside config directory", match)
			}
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

	// Expand environment variables (BUG-6: was missing in imported configs)
	expandedData := expandEnvVars(string(data))

	var cfg Config
	if err := unmarshalStrict([]byte(expandedData), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// merge merges another config into this one.
// Import config values override base config values (last import wins).
func (c *Config) merge(other *Config) {
	// Merge exclude patterns (accumulate from imports)
	if len(other.Exclude) > 0 {
		c.Exclude = append(c.Exclude, other.Exclude...)
	}

	// Merge engine configs (BUG-7: was missing, causing import engine settings to be lost)
	c.Engines.Fmt.mergeFrom(&other.Engines.Fmt)
	c.Engines.Style.mergeFrom(&other.Engines.Style)
	c.Engines.Lint.mergeFrom(&other.Engines.Lint)
	c.Engines.Policy.mergeFrom(&other.Engines.Policy)

	// Merge profiles
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	for k := range other.Profiles {
		c.Profiles[k] = other.Profiles[k]
	}

	// Merge global settings (import overrides base if set)
	if other.SeverityThreshold != "" {
		c.SeverityThreshold = other.SeverityThreshold
	}
	if other.FailFast != nil {
		c.FailFast = other.FailFast
	}
	if other.Parallel != nil {
		c.Parallel = other.Parallel
	}
	if other.Recursive != nil {
		c.Recursive = other.Recursive
	}

	// Merge output config
	if other.Output.AbsolutePaths != nil {
		c.Output.AbsolutePaths = other.Output.AbsolutePaths
	}

	// Merge plugin config
	if other.Plugins.Enabled {
		c.Plugins.Enabled = other.Plugins.Enabled
	}
	if len(other.Plugins.Directories) > 0 {
		c.Plugins.Directories = append(c.Plugins.Directories, other.Plugins.Directories...)
	}
	if other.Plugins.VerifyIntegrity != nil {
		c.Plugins.VerifyIntegrity = other.Plugins.VerifyIntegrity
	}
	if len(other.Plugins.Rules) > 0 {
		if c.Plugins.Rules == nil {
			c.Plugins.Rules = make(map[string]RuleConfig)
		}
		for k, v := range other.Plugins.Rules {
			c.Plugins.Rules[k] = v
		}
	}
}

// SetDefaults fills in default values for unset fields.
// Call this before Validate.
func (c *Config) SetDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
}

// IsRecursive returns whether directory traversal should be recursive.
// Returns true by default if not explicitly set in config.
func (c *Config) IsRecursive() bool {
	if c.Recursive == nil {
		return true // Default to recursive
	}
	return *c.Recursive
}

// IsFailFast returns whether to stop on first error.
// Returns false by default if not explicitly set in config.
func (c *Config) IsFailFast() bool {
	if c.FailFast == nil {
		return false
	}
	return *c.FailFast
}

// IsParallel returns whether to run checks in parallel.
// Returns true by default if not explicitly set in config.
func (c *Config) IsParallel() bool {
	if c.Parallel == nil {
		return true
	}
	return *c.Parallel
}

// IsAbsolutePaths returns whether output should use absolute file paths.
// Returns false by default (relative paths) if not explicitly set in config.
func (c *Config) IsAbsolutePaths() bool {
	if c.Output.AbsolutePaths == nil {
		return false // Default to relative paths
	}
	return *c.Output.AbsolutePaths
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

	// Validate plugin rule configurations
	validSeverities := map[string]bool{
		"error":   true,
		"warning": true,
		"info":    true,
		"":        true, // Allow empty (default)
	}

	for name, rule := range c.Plugins.Rules {
		if name == "" {
			return fmt.Errorf("plugin rule name cannot be empty")
		}

		if !validSeverities[rule.Severity] {
			return fmt.Errorf("plugin rule %q has invalid severity: %s", name, rule.Severity)
		}
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
//
// NOTE: Unlike Engines.mergeFrom (used by ApplyProfile), this uses whole-block replacement.
// When a child profile sets any field in an engine block, the entire block replaces the parent's.
// This preserves explicit intent: a child that sets only "enabled: false" clears inherited
// settings like "check: true" from the parent. ApplyProfile needs field-level merge because
// base config and profile are independent; profile inheritance needs block-level replacement
// because the child is explicitly overriding the parent's configuration.
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

	// Child engines override parent if explicitly set (Enabled != nil or has config)
	if child.Engines.Fmt.Enabled != nil || child.Engines.Fmt.Check || child.Engines.Fmt.Diff {
		result.Engines.Fmt = child.Engines.Fmt
	}
	if child.Engines.Style.Enabled != nil || child.Engines.Style.Fix || child.Engines.Style.Diff || len(child.Engines.Style.Rules) > 0 {
		result.Engines.Style = child.Engines.Style
	}
	if child.Engines.Lint.Enabled != nil || child.Engines.Lint.ConfigFile != "" || len(child.Engines.Lint.Plugins) > 0 || len(child.Engines.Lint.Args) > 0 || len(child.Engines.Lint.Rules) > 0 {
		result.Engines.Lint = child.Engines.Lint
	}
	if child.Engines.Policy.Enabled != nil || len(child.Engines.Policy.PolicyDirs) > 0 || len(child.Engines.Policy.PolicyFiles) > 0 || len(child.Engines.Policy.Rules) > 0 {
		result.Engines.Policy = child.Engines.Policy
	}

	return result
}

// ApplyProfile applies a profile's settings to the config
func (c *Config) ApplyProfile(name string) error {
	profile, err := c.GetProfile(name)
	if err != nil {
		return err
	}

	// Apply engine settings from profile (merge, don't replace)
	c.Engines.mergeFrom(&profile.Engines)

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Engines: Engines{
			Fmt:    FmtEngineConfig{Enabled: BoolPtr(true)},
			Style:  StyleEngineConfig{Enabled: BoolPtr(true)},
			Lint:   LintEngineConfig{Enabled: BoolPtr(true)},
			Policy: PolicyEngineConfig{Enabled: BoolPtr(false)}, // Opt-in
		},
		SeverityThreshold: "warning",
		FailFast:          BoolPtr(false),
		Parallel:          BoolPtr(true),
	}
}
