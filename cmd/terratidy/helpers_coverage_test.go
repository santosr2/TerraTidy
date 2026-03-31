package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/santosr2/terratidy/internal/config"
)

func TestGetEffectiveParallel(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		cliParallel bool
		want        bool
	}{
		{"cli flag true overrides config", &config.Config{Parallel: false}, true, true},
		{"cli flag false uses config true", &config.Config{Parallel: true}, false, true},
		{"both false", &config.Config{Parallel: false}, false, false},
		{"nil config cli false", nil, false, false},
		{"nil config cli true", nil, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getEffectiveParallel(tt.cfg, tt.cliParallel))
		})
	}
}

func TestShouldFailFast(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want bool
	}{
		{"nil config", nil, false},
		{"fail fast enabled", &config.Config{FailFast: true}, true},
		{"fail fast disabled", &config.Config{FailFast: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldFailFast(tt.cfg))
		})
	}
}

func TestIsEngineEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Engines.Fmt.Enabled = true
	cfg.Engines.Style.Enabled = false
	cfg.Engines.Lint.Enabled = true
	cfg.Engines.Policy.Enabled = false

	tests := []struct {
		name   string
		cfg    *config.Config
		engine string
		want   bool
	}{
		{"nil config returns true", nil, "fmt", true},
		{"fmt enabled", cfg, "fmt", true},
		{"style disabled", cfg, "style", false},
		{"lint enabled", cfg, "lint", true},
		{"policy disabled", cfg, "policy", false},
		{"unknown engine returns true", cfg, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isEngineEnabled(tt.cfg, tt.engine))
		})
	}
}

func TestGetEngineConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Engines.Fmt.Config = map[string]any{"indent": 2}
	cfg.Engines.Style.Config = map[string]any{"fix": true}

	tests := []struct {
		name   string
		cfg    *config.Config
		engine string
		want   map[string]any
	}{
		{"nil config", nil, "fmt", nil},
		{"fmt config", cfg, "fmt", map[string]any{"indent": 2}},
		{"style config", cfg, "style", map[string]any{"fix": true}},
		{"lint nil config", cfg, "lint", nil},
		{"unknown engine", cfg, "unknown", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getEngineConfig(tt.cfg, tt.engine))
		})
	}
}
