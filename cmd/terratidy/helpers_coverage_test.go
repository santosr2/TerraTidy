package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/santosr2/TerraTidy/internal/config"
)

func TestGetEffectiveParallel(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		cliParallel   bool
		cliNoParallel bool
		want          bool
	}{
		{"cli flag true overrides config", &config.Config{Parallel: config.BoolPtr(false)}, true, false, true},
		{"cli flag false uses config true", &config.Config{Parallel: config.BoolPtr(true)}, false, false, true},
		{"both false", &config.Config{Parallel: config.BoolPtr(false)}, false, false, false},
		{"nil config cli false", nil, false, false, false},
		{"nil config cli true", nil, true, false, true},
		{"no-parallel overrides config true", &config.Config{Parallel: config.BoolPtr(true)}, false, true, false},
		{"no-parallel overrides --parallel", &config.Config{Parallel: config.BoolPtr(false)}, true, true, false},
		{"no-parallel with nil config", nil, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getEffectiveParallel(tt.cfg, tt.cliParallel, tt.cliNoParallel))
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
		{"fail fast enabled", &config.Config{FailFast: config.BoolPtr(true)}, true},
		{"fail fast disabled", &config.Config{FailFast: config.BoolPtr(false)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldFailFast(tt.cfg))
		})
	}
}

func TestIsEngineEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Engines.Fmt.Enabled = config.BoolPtr(true)
	cfg.Engines.Style.Enabled = config.BoolPtr(false)
	cfg.Engines.Lint.Enabled = config.BoolPtr(true)
	cfg.Engines.Policy.Enabled = config.BoolPtr(false)

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
