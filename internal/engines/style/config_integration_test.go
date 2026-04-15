package style

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/santosr2/TerraTidy/internal/config"
	"github.com/santosr2/TerraTidy/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStyleEngineWithConfiguredRule(t *testing.T) {
	// Create temp file with "this" resource name
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	err := os.WriteFile(tfFile, []byte(`resource "aws_cloudfront_public_key" "this" {
  name = "test"
}`), 0o644)
	require.NoError(t, err)

	// Create style config with the rule enabled
	cfg := &Config{
		Rules: map[string]RuleConfig{
			"style.resource-name-matches-type": {
				Enabled:  config.BoolPtr(true),
				Severity: "info",
			},
		},
	}

	engine := New(cfg)

	findings, err := engine.Run(context.Background(), []string{tfFile})
	require.NoError(t, err)

	t.Logf("Found %d findings", len(findings))
	for _, f := range findings {
		t.Logf("- [%s] %s: %s", f.Severity, f.Rule, f.Message)
	}

	// Should find at least one finding about "this" being generic
	var foundGenericNameFinding bool
	for _, f := range findings {
		if f.Rule == "style.resource-name-matches-type" {
			foundGenericNameFinding = true
			t.Logf("Found expected finding: %s", f.Message)
		}
	}
	require.True(t, foundGenericNameFinding, "Expected to find generic name 'this' finding")
}

func TestStyleEngineAppliesSeverityOverride(t *testing.T) {
	// Create temp file with "this" resource name
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	err := os.WriteFile(tfFile, []byte(`resource "aws_instance" "this" {
  ami = "ami-123"
}`), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name             string
		configSeverity   string
		expectedSeverity sdk.Severity
	}{
		{
			name:             "override to warning",
			configSeverity:   "warning",
			expectedSeverity: sdk.SeverityWarning,
		},
		{
			name:             "override to error",
			configSeverity:   "error",
			expectedSeverity: sdk.SeverityError,
		},
		{
			name:             "override to info",
			configSeverity:   "info",
			expectedSeverity: sdk.SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Rules: map[string]RuleConfig{
					"style.resource-name-matches-type": {
						Enabled:  config.BoolPtr(true),
						Severity: tt.configSeverity,
					},
				},
			}

			engine := New(cfg)
			findings, err := engine.Run(context.Background(), []string{tfFile})
			require.NoError(t, err)

			// Find the resource-name-matches-type finding
			var found bool
			for _, f := range findings {
				if f.Rule == "style.resource-name-matches-type" {
					found = true
					assert.Equal(t, tt.expectedSeverity, f.Severity,
						"severity should be overridden to %s", tt.configSeverity)
				}
			}
			require.True(t, found, "Expected to find resource-name-matches-type finding")
		})
	}
}
