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
//
// Plugin authors should import github.com/santosr2/TerraTidy/pkg/plugins for
// the public plugin API types.
package plugins

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	pkgplugins "github.com/santosr2/TerraTidy/pkg/plugins"
	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// ManifestFileName is the name of the checksum manifest file for plugin verification.
const ManifestFileName = ".terratidy-plugins.sha256"

// relativePath converts an absolute path to a relative path for error messages.
// If conversion fails, returns the original path.
func relativePath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

// PluginType is an alias to pkgplugins.PluginType for internal use.
type PluginType = pkgplugins.PluginType

// PluginMetadata is an alias to pkgplugins.PluginMetadata for internal use.
type PluginMetadata = pkgplugins.PluginMetadata

// RulePlugin is an alias to pkgplugins.RulePlugin for internal use.
type RulePlugin = pkgplugins.RulePlugin

// EnginePlugin is an alias to pkgplugins.EnginePlugin for internal use.
type EnginePlugin = pkgplugins.EnginePlugin

// FormatterPlugin is an alias to pkgplugins.FormatterPlugin for internal use.
type FormatterPlugin = pkgplugins.FormatterPlugin

// Re-export constants from pkg/plugins.
const (
	PluginTypeRule      = pkgplugins.PluginTypeRule
	PluginTypeEngine    = pkgplugins.PluginTypeEngine
	PluginTypeFormatter = pkgplugins.PluginTypeFormatter
)

// Plugin represents a loaded plugin
type Plugin struct {
	Metadata PluginMetadata
	Instance any
}

// Manager manages plugin loading and registration
type Manager struct {
	plugins         map[string]*Plugin
	rules           map[string]sdk.Rule
	engines         map[string]EnginePlugin
	formatters      map[string]FormatterPlugin
	mu              sync.RWMutex
	directories     []string
	verifyIntegrity bool
	logger          *log.Logger
}

// NewManager creates a new plugin manager.
// If verifyIntegrity is true, plugins will be checked against SHA256 manifests.
func NewManager(directories []string, verifyIntegrity bool) *Manager {
	return &Manager{
		plugins:         make(map[string]*Plugin),
		rules:           make(map[string]sdk.Rule),
		engines:         make(map[string]EnginePlugin),
		formatters:      make(map[string]FormatterPlugin),
		directories:     directories,
		verifyIntegrity: verifyIntegrity,
		logger:          log.New(os.Stderr, "terratidy-plugins: ", log.LstdFlags),
	}
}

// SetLogger sets a custom logger for the plugin manager.
func (m *Manager) SetLogger(logger *log.Logger) {
	m.logger = logger
}

// loadManifest loads SHA256 checksums from a manifest file.
// The manifest format is: <sha256hex>  <filename> (like sha256sum output).
// Returns a map of filename -> expected SHA256 hash.
func loadManifest(manifestPath string) (map[string]string, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck // read-only operation

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Format: <hash>  <filename> (two spaces between hash and filename)
		// Also accept single space for compatibility
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid manifest line %d: expected '<hash>  <filename>'", lineNum)
		}

		hash := strings.ToLower(parts[0])
		filename := parts[len(parts)-1] // Last field is filename

		// Validate hash format (64 hex chars for SHA256)
		if len(hash) != 64 {
			return nil, fmt.Errorf("invalid hash on line %d: expected 64 hex chars, got %d", lineNum, len(hash))
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return nil, fmt.Errorf("invalid hex hash on line %d: %w", lineNum, err)
		}

		checksums[filename] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	return checksums, nil
}

// computeFileHash computes the SHA256 hash of a file.
func computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck // read-only operation

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// verifyPluginChecksum verifies a plugin file against the manifest.
// Returns nil if verification passes or is skipped.
// Returns an error if verification fails (checksum mismatch).
// Logs a warning if manifest is missing (warn-only mode for first release).
func (m *Manager) verifyPluginChecksum(pluginPath string, checksums map[string]string) error {
	filename := filepath.Base(pluginPath)

	expectedHash, found := checksums[filename]
	if !found {
		// Plugin not in manifest - warn but allow (for gradual adoption)
		m.logger.Printf("[WARN] plugin %q not found in manifest, skipping verification", filename)
		return nil
	}

	actualHash, err := computeFileHash(pluginPath)
	if err != nil {
		return fmt.Errorf("computing hash for %s: %w", filename, err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
	}

	return nil
}

// LoadAll loads all plugins from the configured directories
func (m *Manager) LoadAll() error {
	for _, dir := range m.directories {
		if err := m.loadFromDirectory(dir); err != nil {
			return fmt.Errorf("loading plugins from %s: %w", relativePath(dir), err)
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
		return fmt.Errorf("%s is not a directory", relativePath(dir))
	}

	// Load checksum manifest if verification is enabled
	var checksums map[string]string
	if m.verifyIntegrity {
		manifestPath := filepath.Join(dir, ManifestFileName)
		var manifestErr error
		checksums, manifestErr = loadManifest(manifestPath)
		if manifestErr != nil {
			if os.IsNotExist(manifestErr) {
				// Manifest doesn't exist - warn but continue (warn-only mode)
				m.logger.Printf("[WARN] no manifest file %s found in %s, plugin verification skipped", ManifestFileName, relativePath(dir))
			} else {
				return fmt.Errorf("loading manifest from %s: %w", relativePath(dir), manifestErr)
			}
		}
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

		switch {
		case strings.HasSuffix(name, ".so"):
			if err := m.loadGoPlugin(path, checksums); err != nil {
				return fmt.Errorf("loading Go plugin %s: %w", name, err)
			}
		case strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml"):
			// Verify YAML rule integrity before loading (if enabled and manifest exists)
			if m.verifyIntegrity && checksums != nil {
				if err := m.verifyPluginChecksum(path, checksums); err != nil {
					return fmt.Errorf("YAML rule verification failed for %s: %w", name, err)
				}
			}
			rule, err := loadYAMLRule(path)
			if err != nil {
				return fmt.Errorf("loading YAML rule %s: %w", name, err)
			}
			// Register rule and plugin atomically to prevent race conditions
			m.mu.Lock()
			m.rules[rule.Name()] = rule
			m.plugins[rule.Name()] = &Plugin{
				Metadata: PluginMetadata{
					Name: rule.Name(),
					Type: PluginTypeRule,
					Path: path,
				},
				Instance: rule,
			}
			m.mu.Unlock()
		case strings.HasSuffix(name, ".sh"):
			if err := m.loadAndRegisterBashRule(path, name, checksums); err != nil {
				return err
			}
		}
	}

	return nil
}

// loadGoPlugin loads a Go plugin from a .so file.
// If checksums is non-nil and verification is enabled, the plugin is verified first.
func (m *Manager) loadGoPlugin(path string, checksums map[string]string) error {
	// Verify plugin integrity before loading (if enabled and manifest exists)
	if m.verifyIntegrity && checksums != nil {
		if err := m.verifyPluginChecksum(path, checksums); err != nil {
			return fmt.Errorf("plugin verification failed for %s: %w", filepath.Base(path), err)
		}
	}

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

	// plugin.Lookup returns a pointer to the symbol, so for a *PluginMetadata
	// variable, we get **PluginMetadata. Dereference to get the actual metadata.
	metaPtr, ok := metaSym.(**PluginMetadata)
	if !ok {
		return fmt.Errorf("PluginMetadata has wrong type (expected **PluginMetadata, got %T)", metaSym)
	}
	metadata := *metaPtr

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
