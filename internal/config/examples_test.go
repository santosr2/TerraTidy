package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isTerraTidyConfig checks if a file looks like a TerraTidy config.
// TerraTidy configs start with "version: 1" near the top, while other YAML files
// in examples/ (GitHub workflows, pre-commit configs, YAML rule definitions) do not.
// We check the first 500 bytes to catch configs with comment preambles while
// excluding files like github-workflow.yaml which have extensive comments before
// any "version:" key that might appear in workflow step definitions.
func isTerraTidyConfig(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	check := string(content)
	if len(check) > 500 {
		check = check[:500]
	}

	return strings.Contains(check, "version:")
}

// findExampleConfigs discovers TerraTidy config files in the examples/ directory.
// It returns absolute paths to all .yaml and .yml files that appear to be TerraTidy configs.
func findExampleConfigs(t *testing.T) []string {
	t.Helper()

	// Find the repo root (examples/ is at repo root)
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	examplesDir := filepath.Join(repoRoot, "examples")
	require.DirExists(t, examplesDir, "examples/ directory should exist")

	// Collect all YAML files at the top level of examples/
	// (yaml-rule/ subdirectory contains rule definitions, not configs)
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	var configFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(examplesDir, name)
		if isTerraTidyConfig(path) {
			configFiles = append(configFiles, path)
		}
	}

	require.NotEmpty(t, configFiles, "should find at least one TerraTidy config in examples/")
	t.Logf("Found %d TerraTidy config files in examples/", len(configFiles))

	return configFiles
}

func TestExampleConfigs_AllParse(t *testing.T) {
	configFiles := findExampleConfigs(t)

	for _, configPath := range configFiles {
		name := filepath.Base(configPath)
		t.Run(name, func(t *testing.T) {
			cfg, err := Load(configPath)
			require.NoError(t, err, "config should parse without error")
			assert.NotNil(t, cfg, "config should not be nil")
		})
	}
}

func TestExampleConfigs_AllValidate(t *testing.T) {
	configFiles := findExampleConfigs(t)

	for _, configPath := range configFiles {
		name := filepath.Base(configPath)
		t.Run(name, func(t *testing.T) {
			cfg, err := Load(configPath)
			require.NoError(t, err, "config should load")

			err = cfg.Validate()
			assert.NoError(t, err, "config should validate without error")
		})
	}
}
