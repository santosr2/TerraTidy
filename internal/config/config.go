package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the complete TerraTidy configuration
type Config struct {
	Version  int      `yaml:"version"`
	Imports  []string `yaml:"imports,omitempty"`
	Engines  Engines  `yaml:"engines"`
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
	Name        string    `yaml:"profile"`
	Description string    `yaml:"description"`
	Inherits    string    `yaml:"inherits,omitempty"`
	Engines     Engines   `yaml:"engines"`
	Overrides   OverridesConfig `yaml:"overrides,omitempty"`
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
	
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
	
	// TODO: Add more validation rules
	
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

