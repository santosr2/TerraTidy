// Package plugins provides a plugin system for extending TerraTidy functionality.
//
// The plugin system supports three types of plugins:
//   - Rule plugins: Custom style/lint rules
//   - Engine plugins: Custom analysis engines
//   - Formatter plugins: Custom output formatters
//
// Plugins are loaded from .so files (Go plugins) that export specific symbols:
//   - PluginMetadata: Plugin information
//   - New: Constructor function returning the plugin instance
//
// Note: The actual .so plugin loading functions (loadGoPlugin, loadRulePlugin,
// loadEnginePlugin, loadFormatterPlugin) are tested via integration tests as
// they require building real compiled plugin binaries.
package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/santosr2/terratidy/pkg/sdk"
)

// PluginType represents the type of plugin
type PluginType string

const (
	// PluginTypeRule represents a custom rule plugin
	PluginTypeRule PluginType = "rule"
	// PluginTypeEngine represents a custom engine plugin
	PluginTypeEngine PluginType = "engine"
	// PluginTypeFormatter represents a custom output formatter plugin
	PluginTypeFormatter PluginType = "formatter"
)

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	Type        PluginType `json:"type"`
	Path        string     `json:"path"`
}

// Plugin represents a loaded plugin
type Plugin struct {
	Metadata PluginMetadata
	Instance interface{}
}

// RulePlugin defines the interface for rule plugins
type RulePlugin interface {
	// GetRules returns all rules provided by this plugin
	GetRules() []sdk.Rule
}

// EnginePlugin defines the interface for engine plugins
type EnginePlugin interface {
	// Name returns the engine name
	Name() string
	// Run executes the engine on the given files
	Run(ctx context.Context, files []string) ([]sdk.Finding, error)
}

// FormatterPlugin defines the interface for formatter plugins
type FormatterPlugin interface {
	// Name returns the formatter name
	Name() string
	// Format formats the findings and writes to the writer
	Format(findings []sdk.Finding, w interface{}) error
}

// Manager manages plugin loading and registration
type Manager struct {
	plugins     map[string]*Plugin
	rules       map[string]sdk.Rule
	engines     map[string]EnginePlugin
	formatters  map[string]FormatterPlugin
	mu          sync.RWMutex
	directories []string
}

// NewManager creates a new plugin manager
func NewManager(directories []string) *Manager {
	return &Manager{
		plugins:     make(map[string]*Plugin),
		rules:       make(map[string]sdk.Rule),
		engines:     make(map[string]EnginePlugin),
		formatters:  make(map[string]FormatterPlugin),
		directories: directories,
	}
}

// LoadAll loads all plugins from the configured directories
func (m *Manager) LoadAll() error {
	for _, dir := range m.directories {
		if err := m.loadFromDirectory(dir); err != nil {
			return fmt.Errorf("loading plugins from %s: %w", dir, err)
		}
	}
	return nil
}

// loadFromDirectory loads all plugins from a directory
func (m *Manager) loadFromDirectory(dir string) error {
	// Expand path
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(home, dir[1:])
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Find plugin files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)

		// Load .so files (Go plugins)
		if strings.HasSuffix(name, ".so") {
			if err := m.loadGoPlugin(path); err != nil {
				return fmt.Errorf("loading Go plugin %s: %w", name, err)
			}
		}
	}

	return nil
}

// loadGoPlugin loads a Go plugin from a .so file
func (m *Manager) loadGoPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("opening plugin: %w", err)
	}

	// Look for plugin metadata
	metaSym, err := p.Lookup("PluginMetadata")
	if err != nil {
		return fmt.Errorf("plugin missing PluginMetadata symbol: %w", err)
	}

	metadata, ok := metaSym.(*PluginMetadata)
	if !ok {
		return fmt.Errorf("PluginMetadata has wrong type")
	}

	metadata.Path = path

	// Load based on plugin type
	switch metadata.Type {
	case PluginTypeRule:
		return m.loadRulePlugin(p, metadata)
	case PluginTypeEngine:
		return m.loadEnginePlugin(p, metadata)
	case PluginTypeFormatter:
		return m.loadFormatterPlugin(p, metadata)
	default:
		return fmt.Errorf("unknown plugin type: %s", metadata.Type)
	}
}

// loadRulePlugin loads a rule plugin
func (m *Manager) loadRulePlugin(p *plugin.Plugin, metadata *PluginMetadata) error {
	sym, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("plugin missing New function: %w", err)
	}

	newFunc, ok := sym.(func() RulePlugin)
	if !ok {
		return fmt.Errorf("new function has wrong signature")
	}

	rulePlugin := newFunc()
	rules := rulePlugin.GetRules()

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, rule := range rules {
		m.rules[rule.Name()] = rule
	}

	m.plugins[metadata.Name] = &Plugin{
		Metadata: *metadata,
		Instance: rulePlugin,
	}

	return nil
}

// loadEnginePlugin loads an engine plugin
func (m *Manager) loadEnginePlugin(p *plugin.Plugin, metadata *PluginMetadata) error {
	sym, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("plugin missing New function: %w", err)
	}

	newFunc, ok := sym.(func() EnginePlugin)
	if !ok {
		return fmt.Errorf("new function has wrong signature")
	}

	engine := newFunc()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.engines[engine.Name()] = engine
	m.plugins[metadata.Name] = &Plugin{
		Metadata: *metadata,
		Instance: engine,
	}

	return nil
}

// loadFormatterPlugin loads a formatter plugin
func (m *Manager) loadFormatterPlugin(p *plugin.Plugin, metadata *PluginMetadata) error {
	sym, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("plugin missing New function: %w", err)
	}

	newFunc, ok := sym.(func() FormatterPlugin)
	if !ok {
		return fmt.Errorf("new function has wrong signature")
	}

	formatter := newFunc()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.formatters[formatter.Name()] = formatter
	m.plugins[metadata.Name] = &Plugin{
		Metadata: *metadata,
		Instance: formatter,
	}

	return nil
}

// GetRules returns all registered rules (including plugin rules)
func (m *Manager) GetRules() map[string]sdk.Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	result := make(map[string]sdk.Rule)
	for k, v := range m.rules {
		result[k] = v
	}
	return result
}

// GetRule returns a specific rule by name
func (m *Manager) GetRule(name string) (sdk.Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, ok := m.rules[name]
	return rule, ok
}

// GetEngines returns all registered engines
func (m *Manager) GetEngines() map[string]EnginePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]EnginePlugin)
	for k, v := range m.engines {
		result[k] = v
	}
	return result
}

// GetEngine returns a specific engine by name
func (m *Manager) GetEngine(name string) (EnginePlugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine, ok := m.engines[name]
	return engine, ok
}

// GetFormatters returns all registered formatters
func (m *Manager) GetFormatters() map[string]FormatterPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]FormatterPlugin)
	for k, v := range m.formatters {
		result[k] = v
	}
	return result
}

// GetFormatter returns a specific formatter by name
func (m *Manager) GetFormatter(name string) (FormatterPlugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	formatter, ok := m.formatters[name]
	return formatter, ok
}

// ListPlugins returns all loaded plugins
func (m *Manager) ListPlugins() []*Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		result = append(result, p)
	}
	return result
}

// RegisterRule registers a rule programmatically
func (m *Manager) RegisterRule(rule sdk.Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.Name()] = rule
}

// RegisterEngine registers an engine programmatically
func (m *Manager) RegisterEngine(engine EnginePlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engines[engine.Name()] = engine
}

// RegisterFormatter registers a formatter programmatically
func (m *Manager) RegisterFormatter(formatter FormatterPlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.formatters[formatter.Name()] = formatter
}
