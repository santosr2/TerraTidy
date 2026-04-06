// Package plugins provides types for TerraTidy plugin development.
//
// This package contains the public API for creating custom TerraTidy plugins.
// Plugin authors should import this package, not internal/plugins.
//
// Three types of plugins are supported:
//   - Rule plugins: Custom style/lint rules
//   - Engine plugins: Custom analysis engines
//   - Formatter plugins: Custom output formatters
//
// Go plugins must export two symbols:
//   - PluginMetadata: A *PluginMetadata variable with plugin info
//   - New: A constructor function returning the plugin instance
package plugins

import (
	"io"

	"github.com/santosr2/TerraTidy/pkg/sdk"
)

// PluginType represents the type of plugin.
type PluginType string

const (
	// PluginTypeRule represents a custom rule plugin.
	PluginTypeRule PluginType = "rule"
	// PluginTypeEngine represents a custom engine plugin.
	PluginTypeEngine PluginType = "engine"
	// PluginTypeFormatter represents a custom output formatter plugin.
	PluginTypeFormatter PluginType = "formatter"
)

// PluginMetadata contains information about a plugin.
// Go plugins must export a variable of this type named "PluginMetadata".
type PluginMetadata struct {
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	Type        PluginType `json:"type"`
	Path        string     `json:"path"` // Set automatically on load
}

// RulePlugin defines the interface for rule plugins.
// Rule plugins provide custom rules for style or lint checking.
type RulePlugin interface {
	// GetRules returns all rules provided by this plugin.
	GetRules() []sdk.Rule
}

// EnginePlugin defines the interface for engine plugins.
// Engine plugins provide custom analysis engines.
type EnginePlugin interface {
	sdk.Engine
}

// FormatterPlugin defines the interface for formatter plugins.
// Formatter plugins provide custom output formats.
type FormatterPlugin interface {
	// Name returns the formatter name (used with --format flag).
	Name() string
	// Format writes findings to the provided writer.
	Format(findings []sdk.Finding, w io.Writer) error
}
